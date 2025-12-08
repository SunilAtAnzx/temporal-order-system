package models

import "time"

// Order represents an order in the system
type Order struct {
	ID        string      `json:"id"`
	Items     []OrderItem `json:"items"`
	Amount    float64     `json:"amount"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// OrderItem represents a single item in an order
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// OrderStatus represents the current status of an order
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusValidated  OrderStatus = "VALIDATED"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusCompleted  OrderStatus = "COMPLETED"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
	OrderStatusExpedited  OrderStatus = "EXPEDITED"
	OrderStatusFailed     OrderStatus = "FAILED"
)

// ValidationRequest represents the request to validate an order
type ValidationRequest struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

// ValidationResponse represents the response from validation service
type ValidationResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// WorkflowState represents the current state of the workflow
type WorkflowState struct {
	OrderID         string      `json:"order_id"`
	Status          OrderStatus `json:"status"`
	ValidationDone  bool        `json:"validation_done"`
	ProcessingDone  bool        `json:"processing_done"`
	PaymentDone     bool        `json:"payment_done"`
	LastUpdated     time.Time   `json:"last_updated"`
}
