package workflows

import (
	"fmt"
	"time"

	"temporal-order-system/activities"
	"temporal-order-system/models"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	SignalCancel   = "cancel"
	SignalExpedite = "expedite"
	QueryState     = "state"
)

// OrderWorkflow is the main workflow for processing orders
func OrderWorkflow(ctx workflow.Context, order models.Order) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("OrderWorkflow started", "order_id", order.ID)

	// Initialize workflow state
	state := models.WorkflowState{
		OrderID:     order.ID,
		Status:      models.OrderStatusPending,
		LastUpdated: workflow.Now(ctx),
	}

	// Setup signal channels
	cancelChan := workflow.GetSignalChannel(ctx, SignalCancel)
	expediteChan := workflow.GetSignalChannel(ctx, SignalExpedite)

	// Setup query handler for workflow state
	err := workflow.SetQueryHandler(ctx, QueryState, func() (models.WorkflowState, error) {
		return state, nil
	})
	if err != nil {
		return fmt.Errorf("failed to set query handler: %w", err)
	}

	// Version handling for backward compatibility
	v := workflow.GetVersion(ctx, "add-payment-processing", workflow.DefaultVersion, 1)

	// Activity options with retry policy
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		HeartbeatTimeout:    5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Create activities instance
	var act *activities.Activities

	// Signal state flags
	cancelled := false
	expedited := false

	// Start async signal handler goroutine
	workflow.Go(ctx, func(gCtx workflow.Context) {
		selector := workflow.NewSelector(gCtx)

		for {
			selector.AddReceive(cancelChan, func(c workflow.ReceiveChannel, more bool) {
				var signal string
				c.Receive(gCtx, &signal)
				cancelled = true
				state.Status = models.OrderStatusCancelled
				state.LastUpdated = workflow.Now(gCtx)
				logger.Info("Order cancelled via signal", "order_id", order.ID)
			})

			selector.AddReceive(expediteChan, func(c workflow.ReceiveChannel, more bool) {
				var signal string
				c.Receive(gCtx, &signal)
				expedited = true
				state.Status = models.OrderStatusExpedited
				state.LastUpdated = workflow.Now(gCtx)
				logger.Info("Order expedited via signal", "order_id", order.ID)
			})

			selector.Select(gCtx)

			// Exit loop if cancelled
			if cancelled {
				break
			}
		}
	})

	// Step 1: Validate Order
	logger.Info("Starting order validation", "order_id", order.ID)
	state.Status = models.OrderStatusPending
	state.LastUpdated = workflow.Now(ctx)

	validateCtx := ctx
	if expedited {
		// Reduce timeout for expedited orders
		validateCtx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
			HeartbeatTimeout:    3 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    500 * time.Millisecond,
				BackoffCoefficient: 2.0,
				MaximumInterval:    5 * time.Second,
				MaximumAttempts:    2,
			},
		})
	}

	err = workflow.ExecuteActivity(validateCtx, act.ValidateOrder, order).Get(ctx, nil)
	if err != nil {
		logger.Error("Order validation failed", "order_id", order.ID, "error", err)
		state.Status = models.OrderStatusFailed
		state.LastUpdated = workflow.Now(ctx)

		// Send notification
		_ = workflow.ExecuteActivity(ctx, act.NotifyCustomer, order, "Order validation failed").Get(ctx, nil)

		return fmt.Errorf("validation failed: %w", err)
	}

	state.ValidationDone = true
	state.Status = models.OrderStatusValidated
	state.LastUpdated = workflow.Now(ctx)
	logger.Info("Order validated successfully", "order_id", order.ID)

	// Check if cancelled
	if cancelled {
		logger.Info("Order processing cancelled after validation", "order_id", order.ID)
		rollbackCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
		})
		_ = workflow.ExecuteActivity(rollbackCtx, act.RollbackOrder, order).Get(ctx, nil)
		return fmt.Errorf("order cancelled by user")
	}

	// Version 1: Add payment processing
	if v >= 1 {
		// Step 2: Process Payment (Child Workflow)
		logger.Info("Starting payment processing", "order_id", order.ID)

		childWorkflowOptions := workflow.ChildWorkflowOptions{
			WorkflowID:               fmt.Sprintf("payment-%s", order.ID),
			WorkflowExecutionTimeout: 2 * time.Minute,
		}
		childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)

		var paymentResult string
		err = workflow.ExecuteChildWorkflow(childCtx, PaymentWorkflow, order).Get(ctx, &paymentResult)
		if err != nil {
			logger.Error("Payment processing failed", "order_id", order.ID, "error", err)
			state.Status = models.OrderStatusFailed
			state.LastUpdated = workflow.Now(ctx)

			// Rollback
			_ = workflow.ExecuteActivity(ctx, act.RollbackOrder, order).Get(ctx, nil)
			_ = workflow.ExecuteActivity(ctx, act.NotifyCustomer, order, "Payment processing failed").Get(ctx, nil)

			return fmt.Errorf("payment failed: %w", err)
		}

		state.PaymentDone = true
		state.LastUpdated = workflow.Now(ctx)
		logger.Info("Payment processed successfully", "order_id", order.ID, "result", paymentResult)
	}

	// Check if cancelled
	if cancelled {
		logger.Info("Order processing cancelled after payment", "order_id", order.ID)
		rollbackCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
		})
		_ = workflow.ExecuteActivity(rollbackCtx, act.RollbackOrder, order).Get(ctx, nil)
		return fmt.Errorf("order cancelled by user")
	}

	// Step 3: Process Order
	logger.Info("Starting order processing", "order_id", order.ID)
	state.Status = models.OrderStatusProcessing
	state.LastUpdated = workflow.Now(ctx)

	processCtx := ctx
	if expedited {
		processCtx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
			HeartbeatTimeout:    3 * time.Second,
		})
	}

	err = workflow.ExecuteActivity(processCtx, act.ProcessOrder, order).Get(ctx, nil)
	if err != nil {
		logger.Error("Order processing failed", "order_id", order.ID, "error", err)
		state.Status = models.OrderStatusFailed
		state.LastUpdated = workflow.Now(ctx)

		// Rollback
		_ = workflow.ExecuteActivity(ctx, act.RollbackOrder, order).Get(ctx, nil)
		_ = workflow.ExecuteActivity(ctx, act.NotifyCustomer, order, "Order processing failed").Get(ctx, nil)

		return fmt.Errorf("processing failed: %w", err)
	}

	state.ProcessingDone = true
	state.Status = models.OrderStatusCompleted
	state.LastUpdated = workflow.Now(ctx)

	// Check if cancelled (though at this point order is already processed)
	if cancelled {
		logger.Info("Order cancellation received but order already completed", "order_id", order.ID)
	}

	// Step 4: Notify Customer
	notificationMessage := "Your order has been processed successfully"
	if expedited {
		notificationMessage = "Your expedited order has been processed successfully"
	}

	err = workflow.ExecuteActivity(ctx, act.NotifyCustomer, order, notificationMessage).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to notify customer", "order_id", order.ID, "error", err)
		// Don't fail the workflow if notification fails
	}

	logger.Info("OrderWorkflow completed successfully", "order_id", order.ID, "expedited", expedited)
	return nil
}
