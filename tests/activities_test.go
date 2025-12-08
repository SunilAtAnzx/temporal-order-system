package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"temporal-order-system/activities"
	"temporal-order-system/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestValidateOrder(t *testing.T) {
	tests := []struct {
		name          string
		order         models.Order
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
		errorContains string
		verifyRequest bool
	}{
		{
			name: "Success - Valid Order",
			order: models.Order{
				ID:     "TEST-001",
				Amount: 500.0,
				Items: []models.OrderItem{
					{
						ProductID: "PROD-001",
						Name:      "Test Product",
						Quantity:  1,
						Price:     500.0,
					},
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				resp := models.ValidationResponse{
					Valid:   true,
					Message: "Order validated successfully",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			},
			wantErr:       false,
			verifyRequest: true,
		},
		{
			name: "Failure - Validation Failed",
			order: models.Order{
				ID:     "TEST-002",
				Amount: 15000.0,
				Items: []models.OrderItem{
					{
						ProductID: "PROD-001",
						Name:      "Expensive Product",
						Quantity:  1,
						Price:     15000.0,
					},
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				resp := models.ValidationResponse{
					Valid:   false,
					Message: "Order amount exceeds limit",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			},
			wantErr:       true,
			errorContains: "validation failed",
		},
		{
			name: "Failure - Server Error",
			order: models.Order{
				ID:     "TEST-003",
				Amount: 500.0,
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			},
			wantErr:       true,
			errorContains: "status 500",
		},
		{
			name: "Failure - Bad Gateway",
			order: models.Order{
				ID:     "TEST-004",
				Amount: 750.0,
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte("Bad Gateway"))
			},
			wantErr:       true,
			errorContains: "status 502",
		},
		{
			name: "Success - Large Order Within Limit",
			order: models.Order{
				ID:     "TEST-005",
				Amount: 9999.0,
				Items: []models.OrderItem{
					{
						ProductID: "PROD-001",
						Name:      "High Value Product",
						Quantity:  10,
						Price:     999.9,
					},
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				resp := models.ValidationResponse{
					Valid:   true,
					Message: "Order validated successfully",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestActivityEnvironment()

			// Create a mock HTTP server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.verifyRequest {
					assert.Equal(t, "/validate", r.URL.Path)
					assert.Equal(t, "POST", r.Method)
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

					var req models.ValidationRequest
					err := json.NewDecoder(r.Body).Decode(&req)
					require.NoError(t, err)
					assert.Equal(t, tt.order.ID, req.OrderID)
					assert.Equal(t, tt.order.Amount, req.Amount)
				}

				tt.mockHandler(w, r)
			}))
			defer mockServer.Close()

			// Create activities with mock server URL
			act := activities.NewActivities(mockServer.URL)
			env.RegisterActivity(act.ValidateOrder)

			// Execute activity
			_, err := env.ExecuteActivity(act.ValidateOrder, tt.order)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessOrder(t *testing.T) {
	tests := []struct {
		name          string
		order         models.Order
		wantErr       bool
		errorContains string
	}{
		{
			name: "Success - Valid Order",
			order: models.Order{
				ID:     "TEST-004",
				Amount: 1500.0,
				Items: []models.OrderItem{
					{
						ProductID: "PROD-001",
						Name:      "Product 1",
						Quantity:  2,
						Price:     500.0,
					},
					{
						ProductID: "PROD-002",
						Name:      "Product 2",
						Quantity:  1,
						Price:     500.0,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Success - Single Item",
			order: models.Order{
				ID:     "TEST-005",
				Amount: 1000.0,
				Items: []models.OrderItem{
					{
						ProductID: "PROD-001",
						Name:      "Product 1",
						Quantity:  2,
						Price:     500.0,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Failure - Amount Mismatch",
			order: models.Order{
				ID:     "TEST-006",
				Amount: 1000.0,
				Items: []models.OrderItem{
					{
						ProductID: "PROD-001",
						Name:      "Product 1",
						Quantity:  2,
						Price:     600.0,
					},
				},
			},
			wantErr:       true,
			errorContains: "amount mismatch",
		},
		{
			name: "Success - Multiple Items Complex",
			order: models.Order{
				ID:     "TEST-007",
				Amount: 2500.0, // 3*250 + 2*500 + 5*150 = 750 + 1000 + 750 = 2500
				Items: []models.OrderItem{
					{
						ProductID: "PROD-001",
						Name:      "Product 1",
						Quantity:  3,
						Price:     250.0,
					},
					{
						ProductID: "PROD-002",
						Name:      "Product 2",
						Quantity:  2,
						Price:     500.0,
					},
					{
						ProductID: "PROD-003",
						Name:      "Product 3",
						Quantity:  5,
						Price:     150.0,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Success - Empty Items",
			order: models.Order{
				ID:     "TEST-008",
				Amount: 0.0,
				Items:  []models.OrderItem{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestActivityEnvironment()

			act := activities.NewActivities("http://localhost:8081")
			env.RegisterActivity(act.ProcessOrder)

			_, err := env.ExecuteActivity(act.ProcessOrder, tt.order)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNotifyCustomer(t *testing.T) {
	tests := []struct {
		name    string
		order   models.Order
		message string
		wantErr bool
	}{
		{
			name: "Success - Order Processed",
			order: models.Order{
				ID:     "TEST-009",
				Amount: 500.0,
			},
			message: "Order processed successfully",
			wantErr: false,
		},
		{
			name: "Success - Order Cancelled",
			order: models.Order{
				ID:     "TEST-010",
				Amount: 750.0,
			},
			message: "Order has been cancelled",
			wantErr: false,
		},
		{
			name: "Success - Payment Failed",
			order: models.Order{
				ID:     "TEST-011",
				Amount: 1200.0,
			},
			message: "Payment authorization failed",
			wantErr: false,
		},
		{
			name: "Success - Empty Message",
			order: models.Order{
				ID:     "TEST-012",
				Amount: 300.0,
			},
			message: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestActivityEnvironment()

			act := activities.NewActivities("http://localhost:8081")
			env.RegisterActivity(act.NotifyCustomer)

			_, err := env.ExecuteActivity(act.NotifyCustomer, tt.order, tt.message)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRollbackOrder(t *testing.T) {
	tests := []struct {
		name    string
		order   models.Order
		wantErr bool
	}{
		{
			name: "Success - Small Order",
			order: models.Order{
				ID:     "TEST-013",
				Amount: 500.0,
			},
			wantErr: false,
		},
		{
			name: "Success - Large Order",
			order: models.Order{
				ID:     "TEST-014",
				Amount: 5000.0,
				Items: []models.OrderItem{
					{
						ProductID: "PROD-001",
						Name:      "Product 1",
						Quantity:  10,
						Price:     500.0,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Success - Zero Amount",
			order: models.Order{
				ID:     "TEST-015",
				Amount: 0.0,
			},
			wantErr: false,
		},
		{
			name: "Success - Complex Order",
			order: models.Order{
				ID:     "TEST-016",
				Amount: 3500.0,
				Items: []models.OrderItem{
					{
						ProductID: "PROD-001",
						Name:      "Product 1",
						Quantity:  5,
						Price:     300.0,
					},
					{
						ProductID: "PROD-002",
						Name:      "Product 2",
						Quantity:  2,
						Price:     1000.0,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestActivityEnvironment()

			act := activities.NewActivities("http://localhost:8081")
			env.RegisterActivity(act.RollbackOrder)

			_, err := env.ExecuteActivity(act.RollbackOrder, tt.order)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
