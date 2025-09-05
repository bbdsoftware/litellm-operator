# E2E Tests for LiteLLM Operator

This directory contains comprehensive end-to-end tests for the LiteLLM Operator, covering all major resources and their interactions.

## Test Structure

### Core Resource Tests

1. **`model_e2e_test.go`** - Tests for Model CRD lifecycle
   - Model creation, update, deletion
   - Invalid parameter handling
   - Multiple models in same namespace

2. **`user_e2e_test.go`** - Tests for User CRD lifecycle
   - User CRUD operations
   - Auto-key creation
   - Email validation
   - Immutable field validation
   - Role enum validation

3. **`team_e2e_test.go`** - Tests for Team CRD lifecycle
   - Team CRUD operations
   - Blocked team handling
   - Budget duration management
   - Multiple teams management
   - Immutable field validation

4. **`virtualkey_e2e_test.go`** - Tests for VirtualKey CRD lifecycle
   - Virtual key CRUD operations
   - User/Team associations
   - Blocked key handling
   - Key expiry and duration
   - Secret management
   - Immutable field validation

### Integration Tests

1. **`integration_e2e_test.go`** - Tests for resource interactions
   - Complete User-Team-VirtualKey workflow
   - TeamMemberAssociation functionality
   - Budget hierarchy validation
   - Multi-user team scenarios
   - Role-based access control
   - Cross-resource dependency validation

## Prerequisites

Before running the e2e tests, ensure you have:

1. **Kind cluster** - For running tests in isolation
2. **PostgreSQL** - Tests automatically deploy CloudNativePG
3. **Cert-manager** - Automatically installed by test setup
4. **Prometheus Operator** - Automatically installed by test setup

## Running the Tests

### Run All E2E Tests

```bash
make test-e2e
```

### Run Specific Test Suites

```bash
# Run only model tests
ginkgo --focus="Model E2E Tests" test/e2e/

# Run only auth-related tests
ginkgo --focus="User E2E Tests|Team E2E Tests|VirtualKey E2E Tests" test/e2e/

# Run only integration tests
ginkgo --focus="Integration E2E Tests" test/e2e/
```

### Run with Verbose Output

```bash
ginkgo -v test/e2e/
```

## Test Data and Cleanup

### Test Namespace

All tests run in the `model-e2e-test` namespace, which is created and destroyed automatically.

### Sample Resources

- `test/samples/test-model-secret.yaml` - Test secret for model authentication
- `test/samples/test-auth-secret.yaml` - Test secret for auth operations
- `test/samples/postgres-secret.yaml` - PostgreSQL connection details
- `test/samples/test-postgres.yaml` - PostgreSQL cluster definition

### Automatic Cleanup

Each test ensures proper cleanup of resources:

- Kubernetes custom resources are deleted
- Associated secrets are cleaned up
- LiteLLM instance data is verified for removal

## Test Patterns

### Resource Lifecycle Testing

1. **Create** - Verify CR creation and LiteLLM registration
2. **Update** - Modify CR and verify LiteLLM synchronisation
3. **Delete** - Remove CR and verify cleanup

### Validation Testing

- Field validation (required, enum, format)
- Immutable field enforcement
- Cross-resource reference validation

### Status Verification

- Ready condition monitoring
- Error condition handling
- Status field population

### Integration Testing

- Resource dependency management
- Budget hierarchy enforcement
- Role-based access control
- Multi-user scenarios

## Debugging Failed Tests

### View Test Logs

```bash
# Get controller logs
kubectl logs -n litellm-operator-system deployment/litellm-operator-controller-manager

# Get specific resource status
kubectl get users,teams,virtualkeys,teammemberassociations -n model-e2e-test -o wide

# Describe specific resource for events
kubectl describe user <user-name> -n model-e2e-test
```

### Manual Resource Inspection

```bash
# Check LiteLLM instance status
kubectl get litellminstance -n model-e2e-test -o yaml

# Verify PostgreSQL is running
kubectl get cluster,pods -n model-e2e-test

# Check secrets
kubectl get secrets -n model-e2e-test
```

### Common Issues

1. **PostgreSQL not ready** - Wait for cluster to be fully initialised
2. **Secret not found** - Ensure test secrets are applied correctly
3. **Controller not responding** - Check controller pod logs for errors
4. **Timeout issues** - Increase test timeout for slower environments

## Contributing

When adding new e2e tests:

1. Follow the existing pattern of lifecycle testing
2. Include both positive and negative test cases
3. Ensure proper cleanup in `AfterEach` or `AfterAll` blocks
4. Add appropriate validation for status conditions
5. Include integration scenarios where applicable
6. Update this README with new test descriptions

## Test Coverage

The e2e tests cover:

- ✅ Model resource lifecycle
- ✅ User resource lifecycle  
- ✅ Team resource lifecycle
- ✅ VirtualKey resource lifecycle
- ✅ TeamMemberAssociation functionality
- ✅ Resource relationship validation
- ✅ Budget hierarchy enforcement
- ✅ Role-based access control
- ✅ Immutable field validation
- ✅ Error condition handling
- ✅ Cleanup verification
