# {{TASK_NAME}} - Task Checklist

**Last Updated**: {{UPDATED_DATE}}
**Status**: Not Started / In Progress / Completed

## Phase 1: {{PHASE_1_NAME}}

- [ ] **Task 1.1**: Write failing tests for [feature/change]
  - File: `path/to/file_test.go`
  - Effort: S/M
  - Dependencies: None
  - Acceptance: Tests written and fail with expected error messages

- [ ] **Task 1.2**: [Implement the actual change]
  - File: `path/to/file.go`
  - Effort: M/L
  - Dependencies: Task 1.1 (tests must exist first)
  - Acceptance: Implementation complete and tests pass

- [ ] **Task 1.3**: [Another task if needed]
  - File: `path/to/file.go`
  - Effort: M
  - Dependencies: Task 1.2
  - Acceptance: [Verification criteria]

### Phase 1 Verification
- [ ] Run: `make test-phase1`
- [ ] Verify: [Manual check]

---

## Phase 2: {{PHASE_2_NAME}}

- [ ] **Task 2.1**: [Task description]
  - File: `path/to/file.go`
  - Effort: L
  - Dependencies: Phase 1 complete
  - Acceptance: [Criteria]

### Phase 2 Verification
- [ ] Run: `make test-phase2`
- [ ] Verify: [Manual check]

---

## Final Verification

### Automated Checks:
- [ ] All tests pass: `make test`
- [ ] Linting passes: `make lint`
- [ ] Build succeeds: `make build`

### Manual Checks:
- [ ] [Manual test 1]
- [ ] [Manual test 2]

## Notes Section

[Space for adding notes during implementation]
