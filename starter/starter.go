package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"temporal-order-system/codec"
	"time"

	"temporal-order-system/models"
	"temporal-order-system/workflows"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

const (
	TaskQueueName = "order-processing-queue"
)

func main() {
	// Command line flags
	orderID := flag.String("order-id", "", "Order ID (optional, auto-generated if not provided)")
	amount := flag.Float64("amount", 1000.0, "Order amount")
	signal := flag.String("signal", "", "Send signal to workflow (cancel or expedite)")
	query := flag.Bool("query", false, "Query workflow state")
	workflowID := flag.String("workflow-id", "", "Workflow ID for signal/query operations")
	flag.Parse()

	// Get Temporal server address from environment or use default
	temporalAddress := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddress == "" {
		temporalAddress = "localhost:7233"
	}

	// Get or generate encryption key
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	var keyBytes []byte
	var err error

	if encryptionKey != "" {
		keyBytes, err = hex.DecodeString(encryptionKey)
		if err != nil {
			log.Fatalf("Failed to decode encryption key: %v", err)
		}
	} else {
		// Generate a random 32-byte key for AES-256
		keyBytes = make([]byte, 32)
		if _, err := rand.Read(keyBytes); err != nil {
			log.Fatalf("Failed to generate encryption key: %v", err)
		}
		log.Printf("Warning: Using generated encryption key. Set ENCRYPTION_KEY env var to match worker.")
		log.Printf("Generated key: %s", hex.EncodeToString(keyBytes))
	}

	// Create data converter with encryption
	dataConverter, err := codec.NewEncryptionDataConverter(keyBytes)
	if err != nil {
		log.Fatalf("Failed to create encryption data converter: %v", err)
	}

	// Create Temporal client with encryption
	c, err := client.Dial(client.Options{
		HostPort:      temporalAddress,
		DataConverter: dataConverter,
	})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Handle signal operations
	if *signal != "" {
		if *workflowID == "" {
			log.Fatal("Workflow ID is required for signal operations. Use -workflow-id flag")
		}
		sendSignal(ctx, c, *workflowID, *signal)
		return
	}

	// Handle query operations
	if *query {
		if *workflowID == "" {
			log.Fatal("Workflow ID is required for query operations. Use -workflow-id flag")
		}
		queryWorkflowState(ctx, c, *workflowID)
		return
	}

	// Start a new workflow
	startWorkflow(ctx, c, *orderID, *amount)
}

func startWorkflow(ctx context.Context, c client.Client, orderID string, amount float64) {
	// Generate order ID if not provided
	if orderID == "" {
		orderID = uuid.New().String()
	}

	// Create order
	order := models.Order{
		ID:     orderID,
		Amount: amount,
		Items: []models.OrderItem{
			{
				ProductID: "PROD-001",
				Name:      "Sample Product 1",
				Quantity:  2,
				Price:     300.0,
			},
			{
				ProductID: "PROD-002",
				Name:      "Sample Product 2",
				Quantity:  1,
				Price:     400.0,
			},
		},
		Status:    models.OrderStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Adjust items to match the specified amount
	// Distribute amount proportionally across all items based on quantity
	if len(order.Items) > 0 {
		totalQuantity := 0
		for _, item := range order.Items {
			totalQuantity += item.Quantity
		}

		if totalQuantity > 0 {
			for i := range order.Items {
				itemProportion := float64(order.Items[i].Quantity) / float64(totalQuantity)
				itemAmount := amount * itemProportion
				order.Items[i].Price = itemAmount / float64(order.Items[i].Quantity)
			}
		}
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("order-workflow-%s", order.ID),
		TaskQueue: TaskQueueName,
	}

	log.Printf("Starting workflow for order: %s", order.ID)
	log.Printf("Order amount: $%.2f", order.Amount)
	log.Printf("Workflow ID: %s", workflowOptions.ID)

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, workflows.OrderWorkflow, order)
	if err != nil {
		log.Fatalf("Unable to execute workflow: %v", err)
	}

	log.Printf("Started workflow successfully")
	log.Printf("WorkflowID: %s", we.GetID())
	log.Printf("RunID: %s", we.GetRunID())
	log.Println("\nTo query workflow state, run:")
	log.Printf("  go run starter/starter.go -query -workflow-id %s\n", we.GetID())
	log.Println("To send signals, run:")
	log.Printf("  go run starter/starter.go -signal expedite -workflow-id %s", we.GetID())
	log.Printf("  go run starter/starter.go -signal cancel -workflow-id %s", we.GetID())

	// Wait for workflow completion (optional)
	log.Println("\nWaiting for workflow to complete...")
	err = we.Get(ctx, nil)
	if err != nil {
		log.Printf("Workflow completed with error: %v", err)
	} else {
		log.Println("Workflow completed successfully!")
	}
}

func sendSignal(ctx context.Context, c client.Client, workflowID, signal string) {
	log.Printf("Sending signal '%s' to workflow: %s", signal, workflowID)

	var signalName string
	switch signal {
	case "cancel":
		signalName = workflows.SignalCancel
	case "expedite":
		signalName = workflows.SignalExpedite
	default:
		log.Fatalf("Unknown signal: %s. Valid signals: cancel, expedite", signal)
	}

	err := c.SignalWorkflow(ctx, workflowID, "", signalName, signal)
	if err != nil {
		log.Fatalf("Failed to send signal: %v", err)
	}

	log.Printf("Signal '%s' sent successfully", signal)
}

func queryWorkflowState(ctx context.Context, c client.Client, workflowID string) {
	log.Printf("Querying workflow state: %s", workflowID)

	resp, err := c.QueryWorkflow(ctx, workflowID, "", workflows.QueryState)
	if err != nil {
		log.Fatalf("Failed to query workflow: %v", err)
	}

	var state models.WorkflowState
	if err := resp.Get(&state); err != nil {
		log.Fatalf("Failed to decode query result: %v", err)
	}

	// Pretty print the state
	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal state: %v", err)
	}

	log.Println("\nWorkflow State:")
	fmt.Println(string(stateJSON))
}
