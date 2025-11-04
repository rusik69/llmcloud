# Testing Guide

## Current Test Coverage

**Overall Coverage: ~50%**

| Package | Coverage | Test Files |
|---------|----------|------------|
| `api/v1alpha1` | ~15% | 5 test files |
| `internal/api` | 78.4% | 1 test file |
| `internal/controller` | ~40% | 4 test files |
| `cmd` | 0.0% | No tests yet |

## Running Tests

```bash
make test                    # Run all unit tests
make test                    # With coverage report
```

Generate HTML coverage report:
```bash
go test ./... -coverprofile cover.out
go tool cover -html=cover.out -o coverage.html
open coverage.html
```

E2E tests:
```bash
make test-e2e
```

## Test Files

### API Types Tests (`api/v1alpha1/`)
- `groupversion_test.go` - GroupVersion registration
- `project_types_test.go` - Project CRD (8 test cases)
- `virtualmachine_types_test.go` - VirtualMachine CRD (9 test cases)
- `llmmodel_types_test.go` - LLMModel CRD (9 test cases)
- `service_types_test.go` - Service CRD (11 test cases)

**Coverage:**
- GroupVersion and scheme registration
- Spec validation
- Object creation and lists
- Status updates and phases
- Resource requirements and quotas
- Member roles and permissions

### API Server Tests (`internal/api/`)
- `server_test.go` - HTTP API handlers (26 test cases)

**Coverage:**
- CORS middleware
- Project, VM, LLMModel, Service CRUD operations
- HTTP method validation
- JSON serialization
- Error handling (404, 400, 405)

### Controller Tests (`internal/controller/`)
- `project_controller_test.go` - Project reconciliation (4 test cases)
- `virtualmachine_controller_test.go` - VM reconciliation (3 test cases)
- `llmmodel_controller_test.go` - LLMModel reconciliation (3 test cases)
- `service_controller_test.go` - Service reconciliation (3 test cases)

**Coverage:**
- Project namespace creation and RBAC setup
- VirtualMachine finalizers and specs
- LLMModel finalizers and replicas
- Service finalizers and configurations
- Role mapping for project members
- Reconciliation loops
- Status updates

## CI Integration

Tests run automatically on every push and pull request via GitHub Actions:
1. Lint - Code quality checks
2. Unit Tests - With coverage reporting
3. Coverage Upload - To Codecov
4. E2E Tests - End-to-end testing with Kind

## Running Individual Package Tests

```bash
# API types
go test ./api/v1alpha1/... -v

# API server
go test ./internal/api/... -v

# Controllers (requires envtest setup)
make setup-envtest
go test ./internal/controller/... -v

# Specific test
go test ./api/v1alpha1/... -run TestProjectCreation -v
```

## Coverage Goals

- **Current:** ~50%
- **Next:** 60% (controller reconciliation logic)
- **Target:** 80% (comprehensive coverage)

## Testing Best Practices

1. **Unit Tests**
   - Test one thing at a time
   - Use table-driven tests
   - Mock external dependencies

2. **Integration Tests**
   - Use fake client for K8s API
   - Test full reconciliation loops
   - Verify status updates

3. **E2E Tests**
   - Test complete workflows
   - Use real Kind cluster
   - Validate CRD behavior

