# {{TASK_NAME}} Implementation Plan

**Created**: {{CREATED_DATE}}
**Last Updated**: {{UPDATED_DATE}}

## Overview

{{OVERVIEW}}

## Current State Analysis

{{CURRENT_STATE}}

### Key Code Locations:
- `path/to/file.go:123-145` - [What's here]
- `path/to/handler.go:67` - [Relevant function]

### Current Implementation Example:
```go
// From pkg/component/handler.go:45-67
func CurrentImplementation() {
    // Show relevant existing code
}
```

## Desired End State

{{DESIRED_STATE}}

## What We're NOT Doing

{{OUT_OF_SCOPE}}

## Implementation Approach

{{APPROACH}}

---

## Phase 1: {{PHASE_1_NAME}}

### Overview
{{PHASE_1_OVERVIEW}}

**TDD Approach**: For each change below, write failing unit tests FIRST to define expected behavior, then implement the code to make tests pass.

### Changes Required:

#### 1. {{COMPONENT_1}}
**File**: `path/to/file.ext:line`
**Changes**: {{CHANGES_DESCRIPTION}}

**Current code:**
```go
// Show what exists now
func OldImplementation() {
    // existing code
}
```

**Proposed changes:**
```go
// Show what needs to change
func NewImplementation() {
    // new code with detailed comments
    // explaining the changes
}
```

**Reasoning**: {{REASONING}}

### Testing for This Phase:

**IMPORTANT: Write failing tests BEFORE implementing code changes (TDD approach)**

1. **First, write the tests** that define expected behavior:
```go
// Example test to add BEFORE implementing the feature
func TestNewFeature(t *testing.T) {
    // This test should FAIL initially
    // test implementation defining expected behavior
}
```

2. **Verify tests fail**: Run tests to confirm they fail for the right reasons
3. **Then implement the code** to make the tests pass
4. **Verify tests pass**: Confirm implementation satisfies the tests

### Success Criteria:

#### Automated Verification:
- [ ] Migration applies cleanly: `make migrate`
- [ ] Unit tests pass: `make test-component`
- [ ] Type checking passes: `go vet ./...`
- [ ] Linting passes: `make lint`
- [ ] Integration tests pass: `make test-integration`

#### Manual Verification:
- [ ] Feature works as expected when tested via [specific method]
- [ ] Performance is acceptable under [conditions]
- [ ] Edge case handling: [specific scenarios]
- [ ] No regressions in [related features]

---

## Testing Strategy

### Unit Tests:
- Test file: `path/to/test_file.go`
- Key scenarios with code examples:

```go
func TestKeyScenario(t *testing.T) {
    // example test
}
```

### Integration Tests:
- [End-to-end scenarios with commands to run]

### Manual Testing Steps:
1. [Specific step to verify feature]
2. [Another verification step]
3. [Edge case to test manually]

## Performance Considerations

{{PERFORMANCE_NOTES}}

## Migration Notes

{{MIGRATION_NOTES}}

## References

- Original ticket: `thoughts/allison/tickets/eng_XXXX.md`
- Key files examined: [list with line ranges]
- Similar patterns found: `[file:line]`
