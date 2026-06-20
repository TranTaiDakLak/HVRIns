# REQUIREMENTS_BACKEND_QA_V1.md

## 1. Document Purpose

This document defines the **backend QA requirements** for the desktop application built with **Go + Wails + Vue**.

The goal of this phase is **not** to validate the full business logic yet. The goal is to validate that the backend foundation is stable, extensible, testable, and ready for later feature development.

This QA phase focuses on:

- backend bootstrap and project structure
- configuration loading and validation
- logging and error handling
- service boundaries and modularity
- plugin/driver registration patterns
- flow runtime foundation
- concurrency safety
- mockability and testability
- stable contracts for Wails/frontend integration

---

## 2. QA Goal

The backend must be verified as a **reliable foundation** so future modules can be added safely, including:

- account processing logic
- flow execution logic
- mail type implementations
- proxy strategies
- request/browser/phone runners
- frontend bridge integration through Wails

The backend is considered ready for the next phase only when:

- the structure is clear
- modules are isolated enough to extend
- concurrency behavior is predictable
- error handling is consistent
- tests cover the core foundation
- future implementations can be added with minimal changes to the core

---

## 3. Scope of This QA Phase

### 3.1 In Scope

This QA phase covers:

1. application bootstrap
2. configuration management
3. logger initialization and structured logging
4. dependency wiring
5. module boundaries
6. interface-based extension points
7. mail driver registry pattern
8. action registry pattern
9. scheduler / worker pool foundation
10. account-level run isolation
11. context timeout / cancellation support
12. mock repository / mock driver support
13. testability of the backend
14. Wails-facing backend service contracts
15. basic observability and diagnostics

### 3.2 Out of Scope

This QA phase does **not** validate the full business behavior of:

- Facebook operations
- real browser automation
- real request automation
- real phone automation
- real proxy network behavior
- production mail provider integration
- persistence layer optimization
- account import data correctness
- UI/UX quality

These will be validated in later phases.

---

## 4. Target Architecture Under QA

The backend must follow a **modular monolith** architecture.

Recommended structure:

```text
backend/
├─ cmd/
│  └─ app/
│     └─ main.go
├─ internal/
│  ├─ bootstrap/
│  ├─ config/
│  ├─ module/
│  │  ├─ account/
│  │  ├─ flow/
│  │  ├─ proxy/
│  │  ├─ mail/
│  │  └─ runtime/
│  ├─ adapter/
│  │  ├─ wails/
│  │  ├─ store/
│  │  └─ runner/
│  └─ shared/
├─ test/
├─ configs/
└─ go.mod
```

### Architecture rules

- business code must live under `internal/module/...`
- adapters must not contain business rules
- modules must communicate through explicit interfaces where replacement is needed
- no God-service or God-package is allowed
- bootstrap must remain readable and explicit
- backend must remain runnable and testable without the real frontend

---

## 5. Functional Quality Requirements

### 5.1 Application Bootstrap

The backend must:

- start successfully with default config
- fail fast with invalid config
- initialize logger before critical services
- wire dependencies explicitly
- expose clean startup and shutdown lifecycle
- support graceful shutdown

### Acceptance

- app starts without panic using sample config
- invalid config returns human-readable error
- shutdown does not leave goroutines hanging

---

### 5.2 Configuration System

The backend must support configuration from a local file such as `configs/app.yaml`.

Config must cover at minimum:

- app name
- app environment
- log level
- runtime max workers
- queue size
- default timeouts
- enabled runners
- enabled mail drivers

### Requirements

- config must be centralized
- config must have default values where reasonable
- config must validate required fields
- invalid values must be rejected early

### Acceptance

- valid config loads successfully
- missing required config fails clearly
- max workers <= 0 is rejected
- unknown log level is rejected or normalized explicitly

---

### 5.3 Structured Logging

The backend must use structured logging.

Minimum log fields:

- timestamp
- level
- message
- module
- operation
- account_id if applicable
- flow_id if applicable
- run_id if applicable
- error if applicable

### Requirements

- no critical backend path may rely on ad-hoc `fmt.Println`
- logger must be injectable into services
- logs must be test-friendly

### Acceptance

- major lifecycle events are logged
- worker start/stop is logged
- job failures are logged with context
- plugin registration failures are logged clearly

---

### 5.4 Error Handling

The backend must use consistent error handling.

### Requirements

- errors must be wrapped with context
- errors must not be swallowed silently
- errors returned to frontend/Wails bridge must be safe and readable
- internal errors must retain enough detail for logs

### Acceptance

- service errors can be traced to source module
- invalid action or mail type produces deterministic error
- timeout errors and cancellation errors are distinguishable

---

### 5.5 Mail Driver Extensibility

The backend must support multiple mail implementations through a registry pattern.

Example future mail types:

- type1
- type2
- type3
- type4

### Requirements

- each mail type must implement a shared contract
- mail types must be registerable without editing core flow logic
- adding a new mail type should require only:
  1. new package
  2. implementation of interface
  3. registration during bootstrap

### Acceptance

- at least 2 mock mail drivers are implemented in test/demo form
- registry can resolve a driver by kind/key
- duplicate registration is rejected clearly
- requesting an unknown mail type returns a controlled error

---

### 5.6 Action Extensibility

Flow actions must also support extension through a registry pattern.

### Requirements

- actions must implement a shared interface
- actions must execute through a flow runner
- flow runner must not depend on action-specific switch-case logic

### Acceptance

- at least 3 mock actions exist for QA:
  - `WAIT`
  - `SET_STATUS`
  - `APPEND_NOTE`
- actions are resolved by key
- unknown action key fails gracefully

---

### 5.7 Flow Runtime Foundation

The backend must support running a flow per account.

### Requirements

- one account execution must be represented as one job/run
- each job must run with its own context
- flow runner must process actions in order
- timeout, retry, and continue-on-error behavior must be supported at the runner level or clearly reserved in the design

### Acceptance

- a mock flow can run for one account successfully
- a mock flow can run for multiple accounts successfully
- one failing action does not crash the whole application

---

### 5.8 Scheduler and Worker Pool

The backend must support concurrent execution using a worker pool.

### Requirements

- concurrency must be bounded by config
- worker count must be configurable
- queue size must be configurable
- jobs must be tracked until completion or cancellation
- the system must not launch unbounded goroutines for batch runs

### Acceptance

- 20 mock account jobs can run with worker limit 5
- total active workers never exceed configured limit
- queue overflow behavior is deterministic and tested

---

### 5.9 Account-Level Isolation

The backend must prevent conflicting runs for the same account.

### Requirements

- account lock or equivalent coordination must exist
- the same account must not run two flows simultaneously unless explicitly designed later

### Acceptance

- when the same account is submitted twice, the second run is rejected, skipped, or queued according to the chosen rule
- the chosen rule must be documented and tested

---

### 5.10 Cancellation and Timeout

The backend must support cancellation and timeout through `context.Context`.

### Requirements

- each job must receive a context
- long-running actions must respect context cancellation
- timeout behavior must be testable

### Acceptance

- cancelling a running batch stops unfinished jobs safely
- a timed-out action returns a timeout-related error
- shutdown triggers cancellation of active work

---

### 5.11 Testability

The backend must be test-first friendly.

### Requirements

- services must be testable without real Wails frontend
- real runners must be replaceable with mocks
- repositories must be replaceable with in-memory mocks
- registry behavior must be unit tested

### Acceptance

- unit tests exist for config validation
- unit tests exist for registry behaviors
- unit tests exist for worker pool behavior
- integration-style tests exist for a mock flow batch

---

### 5.12 Wails Integration Readiness

Even though frontend business integration is not the focus of this QA phase, the backend must be ready to serve Wails bindings later.

### Requirements

- backend app services intended for frontend must expose clean public methods
- frontend-facing methods must use DTOs or stable contract structs
- backend must avoid leaking internal storage models directly where avoidable
- errors returned to frontend must be safe for display

### Acceptance

- at least one demo service method is exposed and callable through a mock/frontend-safe contract
- DTOs are separated from internal execution state where needed

---

## 6. Non-Functional Quality Requirements

### 6.1 Maintainability

- code must be readable by a Go developer without prior project context
- package responsibilities must be obvious
- no circular imports
- no deep hidden magic wiring

### 6.2 Extensibility

The system must allow future addition of:

- new mail types
- new action types
- new runner types
- new storage adapters
- new diagnostics endpoints or commands

without rewriting the core runtime.

### 6.3 Reliability

- batch failures must not crash the whole app
- malformed jobs must fail safely
- invalid driver/action configuration must not cause undefined runtime behavior

### 6.4 Observability

- there must be enough logs to diagnose startup, registration, scheduling, execution, and failure
- logs must allow correlation of flow run issues at account level

### 6.5 Performance Baseline

The goal is not production-scale benchmarking yet, but the backend must demonstrate stable behavior under moderate mock load.

Baseline QA load target:

- 100 mock jobs
- worker limit 10
- no deadlock
- no goroutine leak
- no unbounded memory growth in the test scenario

---

## 7. Required QA Deliverables

The implementation team must provide:

1. backend source scaffold
2. sample config file
3. testable bootstrap
4. logger setup
5. mail registry with mock drivers
6. action registry with mock actions
7. scheduler + worker pool + account lock
8. mock repositories
9. unit tests
10. integration test(s)
11. backend QA notes in markdown
12. a short architecture summary document

---

## 8. Required Tests

### 8.1 Unit Tests

Must include at minimum:

- config load + validation
- logger creation
- mail registry registration + resolve
- action registry registration + resolve
- worker pool scheduling behavior
- account lock behavior
- flow runner step ordering
- timeout / cancellation behavior

### 8.2 Integration Tests

Must include at minimum:

- startup with valid config
- startup failure with invalid config
- batch run across multiple mock accounts
- duplicate account submission behavior
- cancellation during active batch

### 8.3 Optional but Recommended Tests

- race detector run
- benchmark for mock batch execution
- goroutine count sanity checks after batch completion

---

## 9. QA Acceptance Criteria

The backend foundation passes QA only if all items below are true:

1. project structure matches the intended modular design
2. app starts and stops cleanly
3. config validation is implemented and tested
4. structured logging is implemented
5. mail registry is implemented and extensible
6. action registry is implemented and extensible
7. worker pool is bounded and tested
8. account-level run isolation exists and is tested
9. cancellation and timeout are supported and tested
10. no critical path relies on hidden globals
11. mock flow batch test passes
12. documentation for backend QA exists

---

## 10. Explicit Constraints

To keep the backend foundation clean, the implementation must obey these constraints:

- do not introduce microservices
- do not introduce distributed queues
- do not introduce a heavy DI framework
- do not use inheritance-style architecture patterns copied from C#
- do not put business logic in Wails adapter code
- do not hardcode mail types into flow logic via giant switch statements
- do not spawn unbounded goroutines per account batch
- do not mix runtime orchestration with storage-specific details

---

## 11. Recommended Implementation Order

1. bootstrap and config
2. logger
3. module structure
4. mail contract + registry
5. action contract + registry
6. runtime scheduler
7. worker pool
8. account lock
9. mock repositories
10. mock flow runner tests
11. Wails-facing demo method
12. documentation and QA report

---

## 12. Final QA Result Format

When the team reports completion, the result must include:

### ✅ Task checklist
A checked markdown checklist of completed QA items.

### 🔧 Summary
Short summary of what was implemented and verified.

### 🧩 Files changed
Clear file tree of new/modified backend files.

### 🧪 Commands run + results
Actual commands executed and short result summary.

### 🧠 Assumptions
Any assumptions made while implementing the backend foundation.

### ⚠️ Risks & follow-ups
Known gaps, risks, and next recommended steps.

---

## 13. Definition of Done

This backend QA phase is complete when the project has a stable, testable backend foundation ready for future feature development, and the team can confidently add:

- new mail types
- new actions
- new runners
- new storage adapters
- frontend integration methods

without refactoring the core runtime architecture.
