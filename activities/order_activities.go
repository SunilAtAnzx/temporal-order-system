package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"temporal-order-system/models"

	"go.temporal.io/sdk/activity"
)

// Activities contains all order processing activities
type Activities struct {
	httpClient        *http.Client
	validationBaseURL string
}

// NewActivities creates a new Activities instance
func NewActivities(validationBaseURL string) *Activities {
	return &Activities{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		validationBaseURL: validationBaseURL,
	}
}

// ValidateOrder validates an order by calling an external validation service
func (a *Activities) ValidateOrder(ctx context.Context, order models.Order) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Validating order", "order_id", order.ID, "amount", order.Amount)

	// Create validation request
	validationReq := models.ValidationRequest{
		OrderID: order.ID,
		Amount:  order.Amount,
	}

	jsonData, err := json.Marshal(validationReq)
	if err != nil {
		return fmt.Errorf("failed to marshal validation request: %w", err)
	}

	// Call validation service
	url := fmt.Sprintf("%s/validate", a.validationBaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Heartbeat to let Temporal know we're still alive
	activity.RecordHeartbeat(ctx, "calling validation service")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call validation service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("validation service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse validation response
	var validationResp models.ValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validationResp); err != nil {
		return fmt.Errorf("failed to decode validation response: %w", err)
	}

	activity.RecordHeartbeat(ctx, "validation response received")

	if !validationResp.Valid {
		return fmt.Errorf("order validation failed: %s", validationResp.Message)
	}

	logger.Info("Order validated successfully", "order_id", order.ID, "message", validationResp.Message)
	return nil
}

// ProcessOrder simulates order processing with business logic
func (a *Activities) ProcessOrder(ctx context.Context, order models.Order) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing order", "order_id", order.ID)

	// Simulate processing time with context-aware wait
	select {
	case <-time.After(2 * time.Second):
		// Processing complete
	case <-ctx.Done():
		return ctx.Err()
	}

	// Record heartbeat
	activity.RecordHeartbeat(ctx, "order processing in progress")

	// Simulate business logic
	logger.Info("Applying business rules", "order_id", order.ID)

	// Calculate total and verify
	var calculatedTotal float64
	for _, item := range order.Items {
		calculatedTotal += item.Price * float64(item.Quantity)
	}

	if calculatedTotal != order.Amount {
		return fmt.Errorf("order amount mismatch: expected %.2f, got %.2f", calculatedTotal, order.Amount)
	}

	// Simulate inventory check with context-aware wait
	select {
	case <-time.After(500 * time.Millisecond):
		// Inventory check complete
	case <-ctx.Done():
		return ctx.Err()
	}

	activity.RecordHeartbeat(ctx, "inventory checked")

	logger.Info("Order processed successfully", "order_id", order.ID)
	return nil
}

// NotifyCustomer sends a notification to the customer
func (a *Activities) NotifyCustomer(ctx context.Context, order models.Order, message string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Notifying customer", "order_id", order.ID, "message", message)

	// Simulate notification delay with context-aware wait
	select {
	case <-time.After(500 * time.Millisecond):
		// Notification sent
	case <-ctx.Done():
		return ctx.Err()
	}

	logger.Info("Customer notified successfully", "order_id", order.ID)
	return nil
}

// RollbackOrder rolls back order processing in case of failure
func (a *Activities) RollbackOrder(ctx context.Context, order models.Order) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Rolling back order", "order_id", order.ID)

	// Simulate rollback operations with context-aware wait
	select {
	case <-time.After(1 * time.Second):
		// Rollback complete
	case <-ctx.Done():
		return ctx.Err()
	}

	logger.Info("Order rolled back successfully", "order_id", order.ID)
	return nil
}
