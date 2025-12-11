# Worker Versioning with Build IDs

This guide explains how to use Worker Versioning with Build IDs to manage workflow code changes without using `workflow.GetVersion()` checks.

## What is Worker Versioning?

Worker Versioning allows you to:
- Deploy new workflow versions without code-level version checks
- Run multiple versions of workers simultaneously
- Gradually migrate workflows to new versions
- Avoid cluttering workflow code with `if/else` version checks

## Changes Made

### 1. Removed `workflow.GetVersion()` from OrderWorkflow

**Before (with GetVersion):**
```go
// Version handling for backward compatibility
v := workflow.GetVersion(ctx, "add-payment-processing", workflow.DefaultVersion, 1)

// ... later in code
if v >= 1 {
    // Step 2: Process Payment (Child Workflow)
    // ... payment processing code
}
```

**After (with Build IDs):**
```go
// Step 2: Process Payment (Child Workflow)
// Always execute payment processing
// Version management handled by worker Build IDs
```

### 2. Updated Worker Configuration

The worker now includes:
```go
const (
    WorkerVersion = "1.1.0" // Semantic versioning
    BuildID       = "1.1.0"  // Build ID for worker versioning
)

w := worker.New(c, TaskQueueName, worker.Options{
    BuildID: buildID, // Enable worker versioning
})
```

## Setup Instructions

### Step 1: Configure Task Queue Versioning

You need to set up the task queue to use versioning. This is done **once** per task queue:

```bash
# Using tctl (Temporal CLI)
tctl task-queue update-build-ids add \
  --task-queue order-processing-queue \
  --build-id 1.0.0

# Make 1.0.0 the default version for new workflows
tctl task-queue update-build-ids promote \
  --task-queue order-processing-queue \
  --build-id 1.0.0
```

For Temporal Cloud, use `tcld`:
```bash
tcld task-queue update-build-ids add \
  --namespace your-namespace \
  --task-queue order-processing-queue \
  --build-id 1.0.0

tcld task-queue update-build-ids promote \
  --namespace your-namespace \
  --task-queue order-processing-queue \
  --build-id 1.0.0
```

### Step 2: Deploy Workers with Different Build IDs

#### Option A: Environment Variable (Recommended for Production)

```bash
# Deploy old version (1.0.0 - without payment processing)
BUILD_ID=1.0.0 ./bin/worker

# Deploy new version (1.1.0 - with payment processing)
BUILD_ID=1.1.0 ./bin/worker
```

#### Option B: Update Code Constant

In `worker/worker.go`, update:
```go
const (
    WorkerVersion = "1.2.0" // New version
    BuildID       = "1.2.0"  // New Build ID
)
```

Then rebuild and deploy:
```bash
make build
./bin/worker
```

### Step 3: Add New Build ID to Task Queue

When deploying a new version:

```bash
# Add the new build ID
tctl task-queue update-build-ids add \
  --task-queue order-processing-queue \
  --build-id 1.1.0

# Promote it to be the default for NEW workflows
tctl task-queue update-build-ids promote \
  --task-queue order-processing-queue \
  --build-id 1.1.0
```

### Step 4: Gradual Migration

After deploying the new version:

1. **Existing workflows** continue running on their original Build ID (1.0.0)
2. **New workflows** start on the latest Build ID (1.1.0)
3. Both worker versions run simultaneously until all old workflows complete

## Deployment Workflow

### Scenario: Deploying Version with Payment Processing

1. **Initial State**: Workers running with `BUILD_ID=1.0.0` (no payment processing)

2. **Add New Version**:
   ```bash
   # Register new build ID
   tctl task-queue update-build-ids add \
     --task-queue order-processing-queue \
     --build-id 1.1.0
   ```

3. **Deploy New Workers**:
   ```bash
   # Start new workers alongside old ones
   BUILD_ID=1.1.0 ./bin/worker
   ```

4. **Promote New Version** (makes it default for new workflows):
   ```bash
   tctl task-queue update-build-ids promote \
     --task-queue order-processing-queue \
     --build-id 1.1.0
   ```

5. **Monitor and Drain Old Workers**:
   - Old workflows continue on 1.0.0 workers
   - New workflows start on 1.1.0 workers
   - Once all 1.0.0 workflows complete, shut down 1.0.0 workers

## Checking Build ID Configuration

```bash
# View current build IDs for a task queue
tctl task-queue get-build-ids --task-queue order-processing-queue

# View detailed versioning info
tctl task-queue describe --task-queue order-processing-queue
```

## Testing Locally

### Terminal 1: Start Worker with Build ID 1.0.0
```bash
BUILD_ID=1.0.0 ENCRYPTION_KEY=90fd4f781d9d000ec8db1b40a45a94f5e32a9a1568d9f33cd45c34453de1ae48 \
./bin/worker
```

### Terminal 2: Configure Task Queue
```bash
# Add and promote 1.0.0
tctl task-queue update-build-ids add \
  --task-queue order-processing-queue \
  --build-id 1.0.0

tctl task-queue update-build-ids promote \
  --task-queue order-processing-queue \
  --build-id 1.0.0
```

### Terminal 3: Start Workflow
```bash
ENCRYPTION_KEY=90fd4f781d9d000ec8db1b40a45a94f5e32a9a1568d9f33cd45c34453de1ae48 \
./bin/starter -order-id "ORD-V1-001" -amount 2000
```

### Terminal 4: Deploy New Version (1.1.0)
```bash
# Add new build ID
tctl task-queue update-build-ids add \
  --task-queue order-processing-queue \
  --build-id 1.1.0

# Start new worker
BUILD_ID=1.1.0 ENCRYPTION_KEY=90fd4f781d9d000ec8db1b40a45a94f5e32a9a1568d9f33cd45c34453de1ae48 \
./bin/worker

# Promote new version
tctl task-queue update-build-ids promote \
  --task-queue order-processing-queue \
  --build-id 1.1.0

# Start new workflow - will use 1.1.0
ENCRYPTION_KEY=90fd4f781d9d000ec8db1b40a45a94f5e32a9a1568d9f33cd45c34453de1ae48 \
./bin/starter -order-id "ORD-V2-001" -amount 2000
```

## Benefits Over GetVersion()

| Aspect | GetVersion() | Worker Versioning |
|--------|--------------|-------------------|
| Code Complexity | Cluttered with if/else checks | Clean, version-agnostic code |
| Deployment | All workflows use same code | Different versions run simultaneously |
| Rollback | Requires code changes | Deploy old Build ID |
| Testing | Must test all version paths | Test each version independently |
| Long-term Maintenance | Version checks accumulate | No version checks needed |

## Best Practices

1. **Semantic Versioning**: Use semantic versioning (e.g., 1.0.0, 1.1.0, 2.0.0) for Build IDs
2. **Gradual Rollout**: Deploy new workers before promoting the Build ID
3. **Monitor Metrics**: Watch workflow completion rates during migration
4. **Keep Old Workers**: Don't shut down old workers until all their workflows complete
5. **Document Changes**: Keep a changelog of what changed in each Build ID
6. **Test Thoroughly**: Test new versions in staging before production

## Troubleshooting

### Worker Not Picking Up Tasks
```bash
# Check if Build ID is registered
tctl task-queue get-build-ids --task-queue order-processing-queue

# Check worker logs for Build ID
# Should see: "Build ID: 1.1.0"
```

### Workflows Using Wrong Version
```bash
# Check which Build ID is default
tctl task-queue describe --task-queue order-processing-queue

# Verify workflow execution details in UI
# Workflow Properties > Build ID
```

### Deprecation Warning
The `BuildID` field may show deprecation warnings in newer SDK versions. This is expected and the field still works correctly. Future SDK versions will provide updated APIs.

## Migration Checklist

- [ ] Remove `workflow.GetVersion()` calls from workflow code
- [ ] Add Build ID constants to worker.go
- [ ] Update worker.Options to include BuildID
- [ ] Configure task queue with initial Build ID
- [ ] Deploy workers with Build ID set
- [ ] Test workflow execution with new worker
- [ ] Document version changes
- [ ] Plan gradual rollout strategy

## References

- [Temporal Worker Versioning Documentation](https://docs.temporal.io/workers#worker-versioning)
- [Build IDs Guide](https://docs.temporal.io/workers#build-ids)
- [Task Queue Versioning](https://docs.temporal.io/tasks#task-queue-versioning)
