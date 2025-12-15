package activities

import (
	"context"
	"fmt"
	"time"

	"temporal-order-system/models"

	"go.temporal.io/sdk/activity"
)

// PaymentActivities contains all payment-related activities
type PaymentActivities struct{}

// NewPaymentActivities creates a new PaymentActivities instance
func NewPaymentActivities() *PaymentActivities {
	return &PaymentActivities{}
}

// AuthorizePayment authorizes a payment for the given order
func (p *PaymentActivities) AuthorizePayment(ctx context.Context, order models.Order) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Authorizing payment", "order_id", order.ID, "amount", order.Amount)

	// Simulate payment authorization processing with context-aware wait
	select {
	case <-time.After(1 * time.Second):
		// Processing complete
	case <-ctx.Done():
		return "", ctx.Err()
	}

	activity.RecordHeartbeat(ctx, "authorizing payment")

	// Simulate payment validation
	if order.Amount <= 0 {
		return "", fmt.Errorf("invalid payment amount: %.2f", order.Amount)
	}

	if order.Amount > 9999 {
		return "", fmt.Errorf("payment amount exceeds authorization limit")
	}

	// Generate deterministic authorization ID based on activity info
	info := activity.GetInfo(ctx)
	authorizationID := fmt.Sprintf("AUTH-%s-%d", order.ID[:8], info.Attempt)

	logger.Info("Payment authorized successfully", "order_id", order.ID, "authorization_id", authorizationID)
	return authorizationID, nil
}

// CapturePayment captures a previously authorized payment
func (p *PaymentActivities) CapturePayment(ctx context.Context, order models.Order, authorizationID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Capturing payment", "order_id", order.ID, "authorization_id", authorizationID)

	// Simulate payment capture processing with context-aware wait
	select {
	case <-time.After(1500 * time.Millisecond):
		// Processing complete
	case <-ctx.Done():
		return "", ctx.Err()
	}

	activity.RecordHeartbeat(ctx, "capturing payment")

	// Validate authorization ID
	if authorizationID == "" {
		return "", fmt.Errorf("invalid authorization ID")
	}

	// Generate deterministic transaction ID based on activity info
	info := activity.GetInfo(ctx)
	transactionID := fmt.Sprintf("TXN-%s-%d", order.ID[:8], info.Attempt)

	logger.Info("Payment captured successfully", "order_id", order.ID, "transaction_id", transactionID)
	return transactionID, nil
}

// VoidAuthorization voids a payment authorization
func (p *PaymentActivities) VoidAuthorization(ctx context.Context, authorizationID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Voiding authorization", "authorization_id", authorizationID)

	// Simulate void processing with context-aware wait
	select {
	case <-time.After(500 * time.Millisecond):
		// Processing complete
	case <-ctx.Done():
		return ctx.Err()
	}

	logger.Info("Authorization voided successfully", "authorization_id", authorizationID)
	return nil
}

// RefundPayment refunds a captured payment
func (p *PaymentActivities) RefundPayment(ctx context.Context, transactionID string, amount float64) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Refunding payment", "transaction_id", transactionID, "amount", amount)

	// Simulate refund processing with context-aware wait
	select {
	case <-time.After(1 * time.Second):
		// Processing complete
	case <-ctx.Done():
		return "", ctx.Err()
	}

	activity.RecordHeartbeat(ctx, "processing refund")

	// Generate deterministic refund ID based on activity info
	info := activity.GetInfo(ctx)
	// Use a substring of transaction ID safely (minimum of 8 chars or full length)
	txnIDPart := transactionID
	if len(transactionID) > 12 {
		txnIDPart = transactionID[4:12] // Skip "TXN-" prefix and take next 8 chars
	}
	refundID := fmt.Sprintf("REFUND-%s-%d", txnIDPart, info.Attempt)

	logger.Info("Refund processed successfully", "transaction_id", transactionID, "refund_id", refundID)
	return refundID, nil
}
