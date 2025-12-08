package activities

import (
	"context"
	"fmt"
	"time"

	"temporal-order-system/models"

	"github.com/google/uuid"
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

	// Simulate payment authorization processing
	time.Sleep(1 * time.Second)
	activity.RecordHeartbeat(ctx, "authorizing payment")

	// Simulate payment validation
	if order.Amount <= 0 {
		return "", fmt.Errorf("invalid payment amount: %.2f", order.Amount)
	}

	if order.Amount > 9999 {
		return "", fmt.Errorf("payment amount exceeds authorization limit")
	}

	// Generate authorization ID
	authorizationID := fmt.Sprintf("AUTH-%s", uuid.New().String()[:8])

	logger.Info("Payment authorized successfully", "order_id", order.ID, "authorization_id", authorizationID)
	return authorizationID, nil
}

// CapturePayment captures a previously authorized payment
func (p *PaymentActivities) CapturePayment(ctx context.Context, order models.Order, authorizationID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Capturing payment", "order_id", order.ID, "authorization_id", authorizationID)

	// Simulate payment capture processing
	time.Sleep(1500 * time.Millisecond)
	activity.RecordHeartbeat(ctx, "capturing payment")

	// Validate authorization ID
	if authorizationID == "" {
		return "", fmt.Errorf("invalid authorization ID")
	}

	// Generate transaction ID
	transactionID := fmt.Sprintf("TXN-%s", uuid.New().String()[:8])

	logger.Info("Payment captured successfully", "order_id", order.ID, "transaction_id", transactionID)
	return transactionID, nil
}

// VoidAuthorization voids a payment authorization
func (p *PaymentActivities) VoidAuthorization(ctx context.Context, authorizationID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Voiding authorization", "authorization_id", authorizationID)

	// Simulate void processing
	time.Sleep(500 * time.Millisecond)

	logger.Info("Authorization voided successfully", "authorization_id", authorizationID)
	return nil
}

// RefundPayment refunds a captured payment
func (p *PaymentActivities) RefundPayment(ctx context.Context, transactionID string, amount float64) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Refunding payment", "transaction_id", transactionID, "amount", amount)

	// Simulate refund processing
	time.Sleep(1 * time.Second)
	activity.RecordHeartbeat(ctx, "processing refund")

	// Generate refund ID
	refundID := fmt.Sprintf("REFUND-%s", uuid.New().String()[:8])

	logger.Info("Refund processed successfully", "transaction_id", transactionID, "refund_id", refundID)
	return refundID, nil
}
