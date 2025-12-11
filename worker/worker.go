package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"temporal-order-system/codec"

	"temporal-order-system/activities"
	"temporal-order-system/workflows"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Version information - update this when deploying new versions
const (
	WorkerVersion = "1.1.0" // Semantic versioning
	BuildID       = "1.1.0" // Build ID for worker versioning
)

const (
	TaskQueueName = "order-processing-queue"
)

func main() {
	// Get Temporal server address from environment or use default
	temporalAddress := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddress == "" {
		temporalAddress = "localhost:7233"
	}

	// Get WireMock URL from environment or use default
	wiremockURL := os.Getenv("WIREMOCK_URL")
	if wiremockURL == "" {
		wiremockURL = "http://localhost:8081"
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
		log.Printf("Generated encryption key: %s", hex.EncodeToString(keyBytes))
		log.Println("Set ENCRYPTION_KEY environment variable to use this key in production")
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

	// Get Build ID from environment or use default
	buildID := os.Getenv("BUILD_ID")
	if buildID == "" {
		buildID = BuildID
	}

	// Create worker with Build ID for worker versioning
	// Note: Worker versioning requires server-side setup via tctl/tcld commands
	// See WORKER_VERSIONING.md for complete setup instructions
	w := worker.New(c, TaskQueueName, worker.Options{
		// Setting BuildID enables worker versioning
		// This allows you to deploy new workflow versions without workflow.GetVersion() checks
		BuildID:                                buildID,
		MaxConcurrentActivityExecutionSize:     100,
		MaxConcurrentWorkflowTaskExecutionSize: 50,
	})

	// Register workflows
	w.RegisterWorkflow(workflows.OrderWorkflow)
	w.RegisterWorkflow(workflows.PaymentWorkflow)

	// Register activities
	orderActivities := activities.NewActivities(wiremockURL)
	w.RegisterActivity(orderActivities.ValidateOrder)
	w.RegisterActivity(orderActivities.ProcessOrder)
	w.RegisterActivity(orderActivities.NotifyCustomer)
	w.RegisterActivity(orderActivities.RollbackOrder)

	paymentActivities := activities.NewPaymentActivities()
	w.RegisterActivity(paymentActivities.AuthorizePayment)
	w.RegisterActivity(paymentActivities.CapturePayment)
	w.RegisterActivity(paymentActivities.VoidAuthorization)
	w.RegisterActivity(paymentActivities.RefundPayment)

	log.Println("Starting Temporal worker...")
	log.Printf("Worker Version: %s", WorkerVersion)
	log.Printf("Build ID: %s", buildID)
	log.Printf("Temporal address: %s", temporalAddress)
	log.Printf("Task queue: %s", TaskQueueName)
	log.Printf("WireMock URL: %s", wiremockURL)
	log.Println("Registered workflows: OrderWorkflow, PaymentWorkflow")
	log.Println("Encryption: Enabled")
	log.Println("Worker Versioning: Enabled (requires server-side task queue configuration)")

	// Start worker
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Unable to start worker: %v", err)
	}
}
