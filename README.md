# Temporal Order Processing System

A comprehensive order processing system built with Temporal demonstrating workflows, activities, signals, queries, child workflows, encryption, and versioning.

## Features

### Basic Requirements
- **Workflow & Activities**: Main workflow with order validation and processing
- **External Service Integration**: WireMock-based validation service
- **Retry Policies & Timeouts**: Configurable retry and timeout policies for all activities
- **Worker Setup**: Fully configured Temporal worker
- **Starter**: Command-line starter for triggering workflows
- **Signals**: Support for order cancellation and expediting
- **Queries**: Query workflow state at any time
- **Unit Tests**: Comprehensive tests with HTTP mocking

### Advanced Features
- **Encryption/Decryption**: AES-256 GCM payload codec for data encryption
- **Child Workflow**: Payment processing as a child workflow
- **Versioning**: Workflow.getVersion for backward compatibility
- **Messaging**: Complete signal/query implementation

## Architecture

```
temporal-order-system/
├── models/              # Domain models and data structures
├── workflows/           # Temporal workflows (main and child)
├── activities/          # Temporal activities
├── worker/             # Temporal worker setup
├── starter/            # Workflow starter/client
├── codec/              # Encryption/decryption codec
├── tests/              # Unit tests
├── config/             # Configuration files
│   └── wiremock/       # WireMock mappings
└── docker-compose.yml  # Docker setup for Temporal and WireMock
```

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Make (optional, for convenience)

## Setup

### 1. Install Dependencies

```bash
go mod tidy
```

### 2. Start Infrastructure

Start Temporal server, PostgreSQL, Temporal UI, and WireMock:

```bash
docker-compose up -d
```

This will start:
- Temporal server on `localhost:7233`
- Temporal Web UI on `http://localhost:8080`
- WireMock on `http://localhost:8081`
- PostgreSQL on `localhost:5432`

Verify services are running:
```bash
docker-compose ps
```

### 3. Start the Worker

The worker registers workflows and activities with Temporal:

```bash
go run worker/worker.go
```

You should see output indicating the worker is running:
```
Starting Temporal worker...
Temporal address: localhost:7233
Task queue: order-processing-queue
Registered workflows: OrderWorkflow, PaymentWorkflow
Encryption: Enabled
```

**Note**: The worker will generate an encryption key on first run. Copy this key for use with the starter.

## Usage

### Starting a Workflow

In a new terminal, start a workflow:

```bash
# Basic usage with default amount ($1000)
go run starter/starter.go

# Custom amount
go run starter/starter.go -amount 5000

# Custom order ID
go run starter/starter.go -order-id ORDER-123 -amount 2500
```

The starter will output the workflow ID and commands for querying and signaling.

### Querying Workflow State

Query the current state of a workflow:

```bash
go run starter/starter.go -query -workflow-id order-workflow-<ORDER_ID>
```

Example output:
```json
{
  "order_id": "abc-123",
  "status": "PROCESSING",
  "validation_done": true,
  "processing_done": false,
  "payment_done": true,
  "last_updated": "2024-01-15T10:30:00Z"
}
```

### Sending Signals

#### Expedite an Order

Speed up order processing with reduced timeouts:

```bash
go run starter/starter.go -signal expedite -workflow-id order-workflow-<ORDER_ID>
```

#### Cancel an Order

Cancel an order and trigger rollback:

```bash
go run starter/starter.go -signal cancel -workflow-id order-workflow-<ORDER_ID>
```

## Key Components

### Workflows

#### OrderWorkflow (workflows/order_workflow.go:27)

Main workflow that orchestrates order processing:
1. Validates order via external service
2. Processes payment (child workflow)
3. Processes order business logic
4. Notifies customer

Features:
- Signal handlers for cancel/expedite
- Query handler for state inspection
- Versioning support
- Activity retry policies
- Timeout configurations

#### PaymentWorkflow (workflows/payment_workflow.go:18)

Child workflow for payment processing:
1. Authorizes payment
2. Captures payment
3. Handles authorization voiding on failure

### Activities

#### Order Activities (activities/order_activities.go)

- **ValidateOrder** (activities/order_activities.go:28): Validates order via WireMock HTTP service
- **ProcessOrder** (activities/order_activities.go:76): Processes order with business logic
- **NotifyCustomer** (activities/order_activities.go:127): Sends customer notifications
- **RollbackOrder** (activities/order_activities.go:139): Rolls back failed orders

#### Payment Activities (activities/payment_activities.go)

- **AuthorizePayment** (activities/payment_activities.go:18): Authorizes payment
- **CapturePayment** (activities/payment_activities.go:48): Captures authorized payment
- **VoidAuthorization** (activities/payment_activities.go:76): Voids authorization
- **RefundPayment** (activities/payment_activities.go:87): Processes refunds

### Encryption

The system uses AES-256-GCM encryption for all workflow data:

```go
// codec/encryption_codec.go:41
// Encode encrypts payloads before storage
// Decode decrypts payloads on retrieval
```

Set encryption key via environment variable:
```bash
export ENCRYPTION_KEY=<64-character-hex-string>
```

### Versioning

Workflows use versioning for backward compatibility:

```go
// workflows/order_workflow.go:57
v := workflow.GetVersion(ctx, "add-payment-processing", workflow.DefaultVersion, 1)
if v >= 1 {
    // New payment processing logic
}
```

## Testing

### Run Unit Tests

```bash
# Run all tests
go test ./tests/... -v

# Run specific test file
go test ./tests/activities_test.go -v

# Run with coverage
go test ./tests/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Coverage

The test suite covers:
- Activity success and failure scenarios
- HTTP client mocking for external services
- Context cancellation handling
- Payment flow end-to-end
- Error conditions and edge cases

## WireMock Configuration

WireMock is configured to validate orders based on amount:

- Amount > 0 and < $10,000: Valid
- Amount ≤ 0: Invalid (negative/zero)
- Amount ≥ $10,000: Invalid (exceeds limit)

Configuration file: `config/wiremock/mappings/validate-order.json`

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TEMPORAL_ADDRESS` | Temporal server address | `localhost:7233` |
| `WIREMOCK_URL` | WireMock server URL | `http://localhost:8081` |
| `ENCRYPTION_KEY` | Hex-encoded 32-byte key | Auto-generated |

## Monitoring

### Temporal Web UI

Access the Temporal Web UI at `http://localhost:8080` to:
- View running and completed workflows
- See workflow history and events
- Inspect activity executions
- Debug workflow issues
- View encrypted payloads

### Worker Logs

The worker outputs detailed logs including:
- Activity execution
- Workflow progress
- Heartbeat signals
- Error messages

## Advanced Usage

### Custom Retry Policies

Modify retry policies in workflows:

```go
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
```

### Workflow Versioning

When adding new steps to existing workflows:

```go
v := workflow.GetVersion(ctx, "change-id", workflow.DefaultVersion, 2)
if v >= 2 {
    // New logic
}
```

### Custom Data Converter

Create custom encryption keys:

```bash
# Generate a new key
openssl rand -hex 32
# Set as environment variable
export ENCRYPTION_KEY=<generated-key>
```

## Troubleshooting

### Worker Connection Issues

If the worker cannot connect to Temporal:
```bash
# Check Temporal is running
docker-compose ps temporal

# Check logs
docker-compose logs temporal
```

### WireMock Issues

If validation fails unexpectedly:
```bash
# Check WireMock logs
docker-compose logs wiremock

# Test WireMock directly
curl -X POST http://localhost:8081/validate \
  -H "Content-Type: application/json" \
  -d '{"order_id":"test","amount":500}'
```

### Encryption Key Mismatch

If you see decryption errors, ensure the worker and starter use the same encryption key:
```bash
# Use the key from worker output
export ENCRYPTION_KEY=<key-from-worker>
```

## Development

### Adding New Activities

1. Add activity method to `activities/order_activities.go`
2. Register in `worker/worker.go`
3. Call from workflow
4. Add unit tests in `tests/`

### Adding New Workflows

1. Create workflow in `workflows/`
2. Register in `worker/worker.go`
3. Add starter logic if needed
4. Update this README

## Production Considerations

1. **Encryption Keys**: Use secure key management (e.g., AWS KMS, HashiCorp Vault)
2. **Database**: Use production-grade database for Temporal persistence
3. **Monitoring**: Integrate with monitoring systems (Prometheus, Datadog)
4. **Logging**: Use structured logging with correlation IDs
5. **Rate Limiting**: Implement rate limiting for external service calls
6. **Secrets**: Never commit secrets or keys to version control

## License

This is a demonstration project for the Temporal coding challenge.

## Resources

- [Temporal Documentation](https://docs.temporal.io/)
- [Temporal Go SDK](https://github.com/temporalio/sdk-go)
- [WireMock](http://wiremock.org/)
