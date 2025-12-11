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
	PaymentWorkflowName = "PaymentWorkflow"
)

// PaymentWorkflow is a child workflow that handles payment processing
func PaymentWorkflow(ctx workflow.Context, order models.Order) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("PaymentWorkflow started", "order_id", order.ID, "amount", order.Amount)

	// Activity options for payment activities
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 20 * time.Second,
		HeartbeatTimeout:    5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Create payment activities instance
	paymentAct := activities.PaymentActivities{}

	// Step 1: Authorize Payment
	logger.Info("Authorizing payment", "order_id", order.ID)
	var authorizationID string
	err := workflow.ExecuteActivity(ctx, paymentAct.AuthorizePayment, order).Get(ctx, &authorizationID)
	if err != nil {
		logger.Error("Payment authorization failed", "order_id", order.ID, "error", err)
		return "", fmt.Errorf("payment authorization failed: %w", err)
	}

	logger.Info("Payment authorized", "order_id", order.ID, "authorization_id", authorizationID)

	// Step 2: Capture Payment
	logger.Info("Capturing payment", "order_id", order.ID)
	var transactionID string
	err = workflow.ExecuteActivity(ctx, paymentAct.CapturePayment, order, authorizationID).Get(ctx, &transactionID)
	if err != nil {
		logger.Error("Payment capture failed", "order_id", order.ID, "error", err)

		// Attempt to void the authorization
		voidCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
		})
		_ = workflow.ExecuteActivity(voidCtx, paymentAct.VoidAuthorization, authorizationID).Get(ctx, nil)

		return "", fmt.Errorf("payment capture failed: %w", err)
	}

	logger.Info("Payment captured successfully", "order_id", order.ID, "transaction_id", transactionID)

	result := fmt.Sprintf("Payment processed successfully. Transaction ID: %s", transactionID)
	return result, nil
}
