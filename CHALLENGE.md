# Temporal Coding Challenge

This challenge is designed to help us understand your experience with Temporal. We expect you to spend approximately 4-6 hours completing it. During the interview, you will be asked to demonstrate your solution and walk us through your approach.

Please upload your code to an accessible repository (e.g., GitHub) for further review. You are welcome to use AI tools to accelerate your work and reduce the time required to complete the challenge.

## Scenario – Order Process Workflow

You are building a simplified Order Processing System using Temporal. The system should:
- Accept an order request.
- Validate the order by calling a mock external service (e.g., WireMock).
- Process the order asynchronously using Temporal workflows and activities.

## Basic Requirements (Core Temporal Concepts)

### Workflow & Activities
Implement a main workflow that:
- Accepts an order request (e.g., `Order{id, items, amount}`).
- Calls an activity to validate the order by making an HTTP request to a mock server (WireMock).
- Calls another activity to process the order (simulate some business logic).
- Ensure proper retry policies and timeouts for activities.

### Worker Setup
- Create a Temporal worker that registers the workflow and activities.
- Provide a simple starter to trigger the workflow.

### Mock Server
- Configure WireMock (or similar) to simulate an external validation API.
- Example: `POST /validate` returns success/failure based on order amount.

### Signals & Queries
- Add a signal to update the order status (e.g., cancel or expedite).
- Add a query to check the current workflow state.

### Unit Tests
- Write unit tests for at least one activity using mocking (e.g., mock HTTP client).

## Advanced Challenge (Bonus Points)

### Encryption/Decryption
- Implement a Payload Codec to encrypt/decrypt workflow inputs and outputs.

### Child Workflow
- Add an optional child workflow for handling payment processing.

### Versioning
- Demonstrate `Workflow.getVersion` for backward compatibility (e.g., add a new step in the workflow safely).

### Messaging
- Use signals to trigger expedited processing and queries to fetch workflow progress.
