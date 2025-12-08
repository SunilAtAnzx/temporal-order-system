package tests

import (
	"testing"

	"temporal-order-system/activities"
	"temporal-order-system/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestAuthorizePayment(t *testing.T) {
	tests := []struct {
		name           string
		order          models.Order
		wantErr        bool
		errorContains  string
		validateResult func(t *testing.T, authID string)
	}{
		{
			name: "Success - Valid Amount",
			order: models.Order{
				ID:     "TEST-PAY-001",
				Amount: 1000.0,
			},
			wantErr: false,
			validateResult: func(t *testing.T, authID string) {
				assert.NotEmpty(t, authID)
				assert.Contains(t, authID, "AUTH-")
			},
		},
		{
			name: "Failure - Zero Amount",
			order: models.Order{
				ID:     "TEST-PAY-002",
				Amount: 0.0,
			},
			wantErr:       true,
			errorContains: "invalid payment amount",
		},
		{
			name: "Failure - Negative Amount",
			order: models.Order{
				ID:     "TEST-PAY-003",
				Amount: -100.0,
			},
			wantErr:       true,
			errorContains: "invalid payment amount",
		},
		{
			name: "Failure - Exceeds Authorization Limit",
			order: models.Order{
				ID:     "TEST-PAY-004",
				Amount: 60000.0,
			},
			wantErr:       true,
			errorContains: "exceeds authorization limit",
		},
		{
			name: "Success - Maximum Valid Amount",
			order: models.Order{
				ID:     "TEST-PAY-005",
				Amount: 49999.0,
			},
			wantErr: false,
			validateResult: func(t *testing.T, authID string) {
				assert.NotEmpty(t, authID)
				assert.Contains(t, authID, "AUTH-")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestActivityEnvironment()

			paymentAct := activities.NewPaymentActivities()
			env.RegisterActivity(paymentAct.AuthorizePayment)

			val, err := env.ExecuteActivity(paymentAct.AuthorizePayment, tt.order)

			// ExecuteActivity itself can fail for some activities
			if err != nil && tt.wantErr {
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}
			require.NoError(t, err)

			var authID string
			err = val.Get(&authID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.validateResult != nil {
					tt.validateResult(t, authID)
				}
			}
		})
	}
}

func TestCapturePayment(t *testing.T) {
	tests := []struct {
		name           string
		order          models.Order
		authID         string
		setupAuth      bool
		wantErr        bool
		errorContains  string
		validateResult func(t *testing.T, txnID string)
	}{
		{
			name: "Success - Valid Authorization",
			order: models.Order{
				ID:     "TEST-PAY-006",
				Amount: 1500.0,
			},
			setupAuth: true,
			wantErr:   false,
			validateResult: func(t *testing.T, txnID string) {
				assert.NotEmpty(t, txnID)
				assert.Contains(t, txnID, "TXN-")
			},
		},
		{
			name: "Failure - Empty Authorization ID",
			order: models.Order{
				ID:     "TEST-PAY-007",
				Amount: 1000.0,
			},
			authID:        "",
			setupAuth:     false,
			wantErr:       true,
			errorContains: "invalid authorization ID",
		},
		{
			name: "Success - Any Non-Empty Authorization ID",
			order: models.Order{
				ID:     "TEST-PAY-008",
				Amount: 1000.0,
			},
			authID:    "ANY-AUTH-ID",
			setupAuth: false,
			wantErr:   false,
			validateResult: func(t *testing.T, txnID string) {
				assert.NotEmpty(t, txnID)
				assert.Contains(t, txnID, "TXN-")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestActivityEnvironment()

			paymentAct := activities.NewPaymentActivities()
			env.RegisterActivity(paymentAct.AuthorizePayment)
			env.RegisterActivity(paymentAct.CapturePayment)

			var authID string
			if tt.setupAuth {
				// First authorize the payment
				val, err := env.ExecuteActivity(paymentAct.AuthorizePayment, tt.order)
				require.NoError(t, err)
				err = val.Get(&authID)
				require.NoError(t, err)
			} else {
				authID = tt.authID
			}

			// Now capture the payment
			val, err := env.ExecuteActivity(paymentAct.CapturePayment, tt.order, authID)

			// ExecuteActivity itself can fail for some activities
			if err != nil && tt.wantErr {
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}
			require.NoError(t, err)

			var txnID string
			err = val.Get(&txnID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.validateResult != nil {
					tt.validateResult(t, txnID)
				}
			}
		})
	}
}

func TestVoidAuthorization(t *testing.T) {
	tests := []struct {
		name          string
		order         models.Order
		authID        string
		setupAuth     bool
		wantErr       bool
		errorContains string
	}{
		{
			name: "Success - Valid Authorization",
			order: models.Order{
				ID:     "TEST-PAY-009",
				Amount: 1000.0,
			},
			setupAuth: true,
			wantErr:   false,
		},
		{
			name:      "Success - Empty Authorization ID (No Validation)",
			authID:    "",
			setupAuth: false,
			wantErr:   false,
		},
		{
			name:      "Success - Any Authorization ID",
			authID:    "SOME-AUTH-ID",
			setupAuth: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestActivityEnvironment()

			paymentAct := activities.NewPaymentActivities()
			env.RegisterActivity(paymentAct.AuthorizePayment)
			env.RegisterActivity(paymentAct.VoidAuthorization)

			var authID string
			if tt.setupAuth {
				// First authorize the payment
				val, err := env.ExecuteActivity(paymentAct.AuthorizePayment, tt.order)
				require.NoError(t, err)
				err = val.Get(&authID)
				require.NoError(t, err)
			} else {
				authID = tt.authID
			}

			// Now void the authorization
			_, err := env.ExecuteActivity(paymentAct.VoidAuthorization, authID)

			// ExecuteActivity itself can fail for some activities
			if err != nil {
				if tt.wantErr && tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				} else if tt.wantErr {
					assert.Error(t, err)
				} else {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRefundPayment(t *testing.T) {
	tests := []struct {
		name           string
		transactionID  string
		amount         float64
		wantErr        bool
		errorContains  string
		validateResult func(t *testing.T, refundID string)
	}{
		{
			name:          "Success - Valid Refund",
			transactionID: "TXN-12345",
			amount:        500.0,
			wantErr:       false,
			validateResult: func(t *testing.T, refundID string) {
				assert.NotEmpty(t, refundID)
				assert.Contains(t, refundID, "REFUND-")
			},
		},
		{
			name:          "Success - Empty Transaction ID (No Validation)",
			transactionID: "",
			amount:        500.0,
			wantErr:       false,
			validateResult: func(t *testing.T, refundID string) {
				assert.NotEmpty(t, refundID)
				assert.Contains(t, refundID, "REFUND-")
			},
		},
		{
			name:          "Success - Zero Amount (No Validation)",
			transactionID: "TXN-12345",
			amount:        0.0,
			wantErr:       false,
			validateResult: func(t *testing.T, refundID string) {
				assert.NotEmpty(t, refundID)
				assert.Contains(t, refundID, "REFUND-")
			},
		},
		{
			name:          "Success - Negative Amount (No Validation)",
			transactionID: "TXN-12345",
			amount:        -100.0,
			wantErr:       false,
			validateResult: func(t *testing.T, refundID string) {
				assert.NotEmpty(t, refundID)
				assert.Contains(t, refundID, "REFUND-")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestActivityEnvironment()

			paymentAct := activities.NewPaymentActivities()
			env.RegisterActivity(paymentAct.RefundPayment)

			val, err := env.ExecuteActivity(paymentAct.RefundPayment, tt.transactionID, tt.amount)

			// ExecuteActivity itself can fail for some activities
			if err != nil && tt.wantErr {
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}
			require.NoError(t, err)

			var refundID string
			err = val.Get(&refundID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.validateResult != nil {
					tt.validateResult(t, refundID)
				}
			}
		})
	}
}

// TestPaymentWorkflow_FullFlow tests the complete payment flow as an integration test
// This is kept separate as it tests the interaction between multiple activities
func TestPaymentWorkflow_FullFlow(t *testing.T) {
	tests := []struct {
		name          string
		order         models.Order
		refundAmount  float64
		wantErr       bool
		errorContains string
	}{
		{
			name: "Success - Complete Flow",
			order: models.Order{
				ID:     "TEST-PAY-FLOW-001",
				Amount: 2500.0,
			},
			refundAmount: 2500.0,
			wantErr:      false,
		},
		{
			name: "Success - Partial Refund",
			order: models.Order{
				ID:     "TEST-PAY-FLOW-002",
				Amount: 5000.0,
			},
			refundAmount: 2500.0,
			wantErr:      false,
		},
		{
			name: "Success - Small Amount",
			order: models.Order{
				ID:     "TEST-PAY-FLOW-003",
				Amount: 50.0,
			},
			refundAmount: 50.0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestActivityEnvironment()

			paymentAct := activities.NewPaymentActivities()
			env.RegisterActivity(paymentAct.AuthorizePayment)
			env.RegisterActivity(paymentAct.CapturePayment)
			env.RegisterActivity(paymentAct.RefundPayment)

			// Step 1: Authorize
			val, err := env.ExecuteActivity(paymentAct.AuthorizePayment, tt.order)
			require.NoError(t, err)
			var authID string
			err = val.Get(&authID)
			require.NoError(t, err)
			require.NotEmpty(t, authID)
			require.Contains(t, authID, "AUTH-")

			// Step 2: Capture
			val, err = env.ExecuteActivity(paymentAct.CapturePayment, tt.order, authID)
			require.NoError(t, err)
			var txnID string
			err = val.Get(&txnID)
			require.NoError(t, err)
			require.NotEmpty(t, txnID)
			require.Contains(t, txnID, "TXN-")

			// Step 3: Refund
			val, err = env.ExecuteActivity(paymentAct.RefundPayment, txnID, tt.refundAmount)
			require.NoError(t, err)
			var refundID string
			err = val.Get(&refundID)
			require.NoError(t, err)
			require.NotEmpty(t, refundID)
			require.Contains(t, refundID, "REFUND-")

			// Verify all IDs are unique
			assert.NotEqual(t, authID, txnID, "Authorization ID and Transaction ID should be different")
			assert.NotEqual(t, authID, refundID, "Authorization ID and Refund ID should be different")
			assert.NotEqual(t, txnID, refundID, "Transaction ID and Refund ID should be different")
		})
	}
}
