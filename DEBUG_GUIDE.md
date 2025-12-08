# Debugging Guide for Temporal Order System

## Prerequisites

1. **Start Docker services**:
   ```bash
   docker-compose up -d
   ```

2. **Verify services are running**:
   ```bash
   docker ps
   # Should see: temporal, temporal-postgresql, temporal-ui, wiremock
   ```

## Debugging Options

### Option 1: GoLand/IntelliJ IDEA (Recommended for your setup)

#### Debug the Worker

1. Open `worker/worker.go`
2. Set breakpoints in your activities or workflow code (e.g., line 35 in `activities/order_activities.go`)
3. Click the green arrow next to `func main()` and select **"Debug 'go build worker.go'"**

OR

Create a run configuration:
- Go to **Run > Edit Configurations**
- Click **+** > **Go Build**
- Name: `Debug Worker`
- Run kind: **File**
- Files: `worker/worker.go`
- Working directory: `$ProjectFileDir$`
- Click **OK** and then click the debug icon

#### Debug the Starter

1. Open `starter/starter.go`
2. Set breakpoints where you want to inspect
3. Click the green arrow next to `func main()` and select **"Debug 'go build starter.go'"**

#### Debug Tests

1. Open any test file (e.g., `tests/activities_test.go`)
2. Click the green arrow next to any test function
3. Select **"Debug 'TestValidateOrder_Success'"**

### Option 2: VS Code

Create `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Worker",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/worker/worker.go",
            "env": {},
            "args": []
        },
        {
            "name": "Debug Starter",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/starter/starter.go",
            "env": {},
            "args": []
        },
        {
            "name": "Debug Current Test",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/tests",
            "env": {},
            "args": ["-test.v"]
        }
    ]
}
```

Usage:
1. Set breakpoints in your code
2. Press `F5` or go to **Run > Start Debugging**
3. Select the configuration you want to run

### Option 3: Delve (Command Line)

#### Debug Worker
```bash
# Install delve if not already installed
go install github.com/go-delve/delve/cmd/dlv@latest

# Start worker in debug mode
dlv debug ./worker/worker.go
```

#### Debug Starter
```bash
dlv debug ./starter/starter.go
```

Common Delve commands:
```
break <file>:<line>     # Set breakpoint
break <function>        # Set breakpoint at function
continue (c)            # Continue execution
next (n)                # Step over
step (s)                # Step into
print <var>             # Print variable
list                    # Show source code
breakpoints (bp)        # List breakpoints
clear <number>          # Clear breakpoint
quit (q)                # Quit debugger
```

## Debugging Strategy for Temporal Applications

### 1. Debug Workflows

**Important**: Temporal workflows have determinism requirements. Breakpoints work, but be aware:

```go
// In workflows/order_workflow.go
func OrderWorkflow(ctx workflow.Context, order models.Order) (*models.OrderResult, error) {
    logger := workflow.GetLogger(ctx)

    // Set breakpoint here to inspect order
    logger.Info("Starting order workflow", "order_id", order.ID)

    // Set breakpoint here to inspect workflow state
    if order.Status == models.StatusCancelled {
        return nil, fmt.Errorf("order already cancelled")
    }

    // Debug activity execution
    err := workflow.ExecuteActivity(ctx, activities.ValidateOrder, order).Get(ctx, nil)
    // Set breakpoint here to check if validation succeeded

    return result, nil
}
```

### 2. Debug Activities

Activities are easier to debug since they're just regular Go functions:

```go
// In activities/order_activities.go
func (a *Activities) ValidateOrder(ctx context.Context, order models.Order) error {
    logger := activity.GetLogger(ctx)

    // Set breakpoint here
    logger.Info("Validating order", "order_id", order.ID)

    // Set breakpoint here to inspect HTTP request
    resp, err := a.httpClient.Do(req)

    // Set breakpoint here to inspect response
    if err != nil {
        return err
    }

    return nil
}
```

### 3. Debug Tests

Test debugging is the easiest:

```bash
# Run specific test with verbose output
go test -v ./tests -run TestValidateOrder_Success

# Debug specific test in GoLand:
# Just click the debug icon next to the test function
```

### 4. Using Temporal Web UI for Debugging

The Temporal Web UI is invaluable for debugging:

1. Open http://localhost:8080
2. Find your workflow execution
3. View:
   - **Event History**: See every workflow event
   - **Input/Output**: Inspect workflow parameters
   - **Stack Trace**: See current execution point
   - **Queries**: Check current state
   - **Pending Activities**: See what's running

## Common Debugging Scenarios

### Scenario 1: Activity Failing

1. Set breakpoint in the activity function
2. Start worker in debug mode
3. Run starter to trigger workflow
4. Step through activity code
5. Check Temporal UI for retry attempts

### Scenario 2: Workflow Not Starting

1. Check worker logs for registration errors
2. Verify task queue name matches between worker and starter
3. Check Temporal server is running: `docker ps`
4. Verify network connectivity

### Scenario 3: Signal Not Received

1. Set breakpoint in signal handler
2. Start worker in debug mode
3. Start workflow
4. Send signal via starter or temporal CLI
5. Step through signal handling code

### Scenario 4: Debugging with WireMock

Check WireMock is returning expected responses:

```bash
# Check WireMock logs
docker logs wiremock

# Test validation endpoint directly
curl -X POST http://localhost:8081/validate \
  -H "Content-Type: application/json" \
  -d '{"orderId": "test-123", "amount": 100.50}'
```

## Debugging Tips

### Enable Verbose Logging

Edit `worker/worker.go` or `starter/starter.go`:

```go
clientOptions := client.Options{
    HostPort: "localhost:7233",
    Logger:   logger,
    // Add connection options for more verbose logging
}
```

### Use Workflow Queries for Live Debugging

While workflow is running, query its state:

```bash
# Using temporal CLI
temporal workflow query \
    --workflow-id order-<uuid> \
    --query-type getOrderState
```

### Inspect Workflow History

```bash
temporal workflow show \
    --workflow-id order-<uuid> \
    --fields long
```

### Remote Debugging

If you need to debug in a container:

```yaml
# Add to docker-compose.yml
worker:
  build: .
  ports:
    - "2345:2345"  # Delve debugging port
  command: dlv debug --headless --listen=:2345 --api-version=2 ./worker/worker.go
```

Then connect from your IDE to `localhost:2345`

## Troubleshooting

### Breakpoints Not Hitting

- Ensure you're running in debug mode, not normal mode
- Check that code has been rebuilt with debug symbols
- Verify the worker is actually executing (check Temporal UI)

### Can't Connect to Temporal

```bash
# Check if Temporal is running
docker ps | grep temporal

# Check Temporal logs
docker logs temporal

# Restart if needed
docker-compose restart temporal
```

### Debugger Timing Out

Temporal has timeout configurations. During debugging, you might hit timeouts:
- Activity Start-to-Close timeout
- Workflow Execution timeout

Consider increasing timeouts during development in your workflow code.
