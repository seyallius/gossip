# Graph Report - /root/gossip  (2026-09-05)

## Corpus Check
- 49 files · ~53,878 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 617 nodes · 1437 edges · 33 communities (32 shown, 1 thin omitted)
- Extraction: 62% EXTRACTED · 37% INFERRED · 0% AMBIGUOUS · INFERRED: 533 edges (avg confidence: 0.82)
- Token cost: 200,719 input · 47,188 output

## Community Hubs (Navigation)
- Event Bus Core Dispatch
- Middleware Benchmarks and Auth Flow
- Batch Processor and Benchmarks
- Event Bus Config and Tests
- Event Filter Package
- Error Constructors and Tests
- Safe Type Casting Utilities
- No-Data and Type Assertion Errors
- Error Predicate Functions
- Service Handler Registration Docs
- Publish and Subscribe Internals
- Event Naming and Testing Guides
- Documentation Site Index
- Fatal, Retryable and Validation Errors
- Typed Error Handling Docs
- Examples Index and Batch API
- Middleware Composition Docs
- Batch Ordering and Shutdown Docs
- Processing and Timeout Errors
- Middleware System Overview
- Event Structure and Metadata Docs
- In-Process Scope and Casting Docs
- Configuration and Lifecycle Guidance
- Invalid Event Error Handling
- Event Filtering Docs
- Global Bus and Shutdown Testing
- Worker Pool and Global Bus
- Unsupported Event Type Errors
- Dispatch and Graceful Degradation
- Test Data Payload Helpers
- Event Metadata Tests
- Batch Shutdown Pitfalls
- Module Definition

## God Nodes (most connected - your core abstractions)
1. `NewEvent()` - 126 edges
2. `NewEventBus()` - 50 edges
3. `EventBus` - 35 edges
4. `NewBatchProcessor()` - 30 edges
5. `Event` - 30 edges
6. `DefaultConfig()` - 23 edges
7. `WithRetry()` - 21 edges
8. `WithTimeout()` - 21 edges
9. `Batch` - 18 edges
10. `NewFilteredProcessor()` - 15 edges

## Surprising Connections (you probably didn't know these)
- `Go CI Build and Test Workflow` --conceptually_related_to--> `Gossip Examples Guide`  [AMBIGUOUS]
  .github/workflows/go.yml → examples/README.md
- `Getting Started Guide` --semantically_similar_to--> `Gossip Event Bus Library`  [INFERRED] [semantically similar]
  docs/getting-started.md → README.md
- `Gossip Documentation Overview` --semantically_similar_to--> `Gossip Event Bus Library`  [INFERRED] [semantically similar]
  docs/README.md → README.md
- `In-Process Scope and Crash Loss` --semantically_similar_to--> `In-Process Only Scope`  [INFERRED] [semantically similar]
  docs/getting-started.md → README.md
- `Middleware Composition Pattern` --semantically_similar_to--> `Middleware Chain Ordering Rule`  [INFERRED] [semantically similar]
  examples/README.md → README.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Middleware Stack Implementing func(EventProcessor) EventProcessor** — implementation_middleware, implementation_withretry, implementation_withtimeout, implementation_withrecovery, implementation_withlogging, implementation_chain [EXTRACTED 1.00]
- **Graceful Drain-on-Shutdown Flow** — implementation_shutdown_ordering, implementation_shutdown_once, implementation_worker_range_over_channel, implementation_publish_readlock_close_race, implementation_batch_shutdown, readme_graceful_shutdown, examples_testing_shutdown_sync, examples_readme_shutdown_pitfall [INFERRED 0.85]
- **Runnable Example Program Suite** — examples_readme_auth_service, examples_readme_ecommerce, examples_readme_microservices, examples_readme_error_handling, examples_readme_type_assertion [EXTRACTED 1.00]
- **SubscribeMultiple Fan-In Processors Across Examples** — docs_examples_auth_service_securityauditprocessor, docs_examples_ecommerce_ordernotificationprocessor, docs_examples_microservices_crossserviceauditprocessor, docs_examples_auth_service_bulk_subscription, docs_implementation_overview_eventbus [INFERRED 0.85]
- **Graceful Shutdown Drain Sequence** — docs_implementation_architecture_shutdown, docs_implementation_architecture_worker, docs_implementation_architecture_publish, docs_implementation_architecture_shutdown_ordering, docs_implementation_architecture_worker_range_not_select, docs_implementation_architecture_publish_readlock_over_send [EXTRACTED 1.00]
- **Typed Error Conditional Handling Flow** — docs_examples_error_handling_emailnotificationprocessor, docs_examples_error_handling_witherrorhandling, docs_examples_error_handling_handleprocessorerror, docs_implementation_api_error_checking_functions, docs_examples_type_assertion_example_handleerror, docs_implementation_api_casteventdata [INFERRED 0.85]

## Communities (33 total, 1 thin omitted)

### Community 0 - "Event Bus Core Dispatch"
Cohesion: 0.06
Nodes (44): LoginData, PasswordChangedData, UserRegisteredData, EventBus, OrderData, OrderService, CancelFunc, Context (+36 more)

### Community 1 - "Middleware Benchmarks and Auth Flow"
Cohesion: 0.12
Nodes (48): AuthService, NewEvent(), BenchmarkWithRetry(), BenchmarkWithTimeout(), B, Chain(), BenchmarkMiddlewareChain(), BenchmarkMiddlewareCombinations() (+40 more)

### Community 2 - "Batch Processor and Benchmarks"
Cohesion: 0.09
Nodes (37): Batch, BatchConfig, BatchErrorHandler, BatchProcessor, BenchmarkBatchProcessor_Add(), BenchmarkBatchProcessor_Flush(), B, BenchmarkBatchProcessor_Add_TableDriven() (+29 more)

### Community 3 - "Event Bus Config and Tests"
Cohesion: 0.11
Nodes (42): Config, BenchmarkEventBus_BufferFull(), BenchmarkEventBus_ConcurrentSubscribePublish(), BenchmarkEventBus_MemoryAllocations_Publish(), BenchmarkEventBus_PublishAsync_NoProcessors(), BenchmarkEventBus_PublishAsync_WithContention(), BenchmarkEventBus_PublishSync_HeavyPayload(), BenchmarkEventBus_Subscribe() (+34 more)

### Community 4 - "Event Filter Package"
Cohesion: 0.13
Nodes (38): And(), BenchmarkAndFilter(), BenchmarkFilterByMetadata(), B, BenchmarkFilterByMetadata_TableDriven(), BenchmarkFilterCombinations(), BenchmarkFilterWithDifferentMetadata(), B (+30 more)

### Community 5 - "Error Constructors and Tests"
Cohesion: 0.16
Nodes (28): AsFatalError(), AsRetryable(), NewFatalError(), NewProcessingError(), NewRetryableError(), NewTimeoutError(), NewValidationError(), T (+20 more)

### Community 6 - "Safe Type Casting Utilities"
Cohesion: 0.12
Nodes (20): T, TestCastEventData_NilEvent(), TestCastEventData_Success(), TestCastEventData_TypeAssertionFails(), TestCastEventData_TypedNil_Default(), TestCastEventData_TypedNil_WithCustomChecker(), TestCastEventData_TypedNil_WithReflect(), TestCastEventData_UntypedNil() (+12 more)

### Community 7 - "No-Data and Type Assertion Errors"
Cohesion: 0.14
Nodes (17): UserData, NoDataError, AsNoDataError(), NewNoDataError(), NewNoDataErrorForField(), NewTypeAssertionError(), TestAsNoDataError(), TestAsTypeAssertionError() (+9 more)

### Community 8 - "Error Predicate Functions"
Cohesion: 0.26
Nodes (15): TypeAssertionError, AsTypeAssertionError(), IsFatal(), IsInvalidEvent(), IsNoData(), IsProcessing(), IsRetryable(), IsTimeout() (+7 more)

### Community 9 - "Service Handler Registration Docs"
Cohesion: 0.15
Nodes (17): Bulk Subscription via SubscribeMultiple, emailNotificationProcessor (auth), LoginData, PasswordChangedData, securityAuditProcessor, setupSecurityHandlers, UserRegisteredData, EmailService (+9 more)

### Community 10 - "Publish and Subscribe Internals"
Cohesion: 0.13
Nodes (17): Async Publishing By Default, Copy-on-Write Removal, Plain Append Addition, EventBus struct fields, Rejecting nil Events at Publish, EventBus.Publish, Read Lock Held Across the Closed-Check and the Send, EventBus.PublishSync, removeSubscriptionAt (+9 more)

### Community 11 - "Event Naming and Testing Guides"
Cohesion: 0.14
Nodes (16): Go CI Build and Test Workflow, Event Naming Conventions, Type Assertion Example, Gossip Testing Guide, Mock Dependency Processor Testing, Table-Driven Processor Tests, any for Event Data Instead of Generics, Functional Options for Nil Checking (+8 more)

### Community 12 - "Documentation Site Index"
Cohesion: 0.22
Nodes (15): GitBook Documentation Config, Getting Started Guide, In-Process Scope and Crash Loss, Event-Driven Solution, Gossip Documentation Overview, Tightly Coupled Side Effects Problem, GitBook Navigation Summary, Auth Service Example (+7 more)

### Community 13 - "Fatal, Retryable and Validation Errors"
Cohesion: 0.17
Nodes (12): FatalError, RetryableError, ValidationError, AsValidationError(), NewFatalErrorf(), NewProcessingErrorf(), NewRetryableErrorf(), NewTimeoutErrorf() (+4 more)

### Community 14 - "Typed Error Handling Docs"
Cohesion: 0.20
Nodes (14): auditLogProcessor (auth), auditLogProcessor (typed errors), Conditional Error Handling by Error Type, emailNotificationProcessor (typed errors), handleProcessorError, UserData (error handling example), BadData, badUserProcessor (+6 more)

### Community 15 - "Examples Index and Batch API"
Cohesion: 0.15
Nodes (14): Authentication Service Example, Hierarchical Event Naming, E-commerce Platform Example, Error Handling Example, crossServiceAuditProcessor, Example Structure Convention, Gossip Examples Index, Type Assertion Error Handling Example (+6 more)

### Community 16 - "Middleware Composition Docs"
Cohesion: 0.21
Nodes (14): Middleware Composition for Robust Processors, Typed Errors Compose With Retry and Timeout Middleware, WithErrorHandling (custom middleware), Chain, Middleware type, ProcessingError, RetryableError, TimeoutError (+6 more)

### Community 17 - "Batch Ordering and Shutdown Docs"
Cohesion: 0.16
Nodes (14): Batch Processing for High-Volume Events, Bus-Before-Batch Shutdown Ordering, batchEmailProcessor, OrderService.CreateOrder, inventoryProcessor, OrderData, orderNotificationProcessor, OrderService (+6 more)

### Community 18 - "Processing and Timeout Errors"
Cohesion: 0.19
Nodes (7): ProcessingError, TimeoutError, AsProcessingError(), AsTimeoutError(), causeText(), joinErrorParts(), TestAsTimeoutError()

### Community 19 - "Middleware System Overview"
Cohesion: 0.21
Nodes (13): Error Handling Example, Middleware Composition Pattern, Middleware Chain, Middleware System, WithLogging Middleware, WithRecovery Middleware, WithRetry Middleware, WithTimeout Middleware (+5 more)

### Community 20 - "Event Structure and Metadata Docs"
Cohesion: 0.20
Nodes (12): AuthService, Event Fan-out, Metadata for Request Tracking, AuthService.RegisterUser, Gossip API Reference, FilterByMetadata, Architecture Deep Dive, Event struct (+4 more)

### Community 21 - "In-Process Scope and Casting Docs"
Cohesion: 0.24
Nodes (11): In-Process Bus, Not a Message Broker, Microservices Communication Example, Shared Event Bus Across Services, FastUserDataNilChecker, Safe Type Assertion Over Panicking, userProcessor, utils.CastEventData, utils.NilChecker (+3 more)

### Community 22 - "Configuration and Lifecycle Guidance"
Cohesion: 0.20
Nodes (11): Config Zero-Value Fallback, Circular Event Dependencies Pitfall, Configuration Tuning by Event Volume, FlushPeriod Ticker Guard, EventBus Coordinator, Non-Blocking Publish with Drop Reporting, RWMutex for the Subscription Map, Subscription Registry (+3 more)

### Community 23 - "Invalid Event Error Handling"
Cohesion: 0.22
Nodes (9): InvalidEventError, AsInvalidEventError(), NewInvalidEventError(), NewInvalidEventErrorf(), TestAsInvalidEventError(), TestInvalidEventError(), TestIsInvalidEvent(), CastEventData() (+1 more)

### Community 24 - "Event Filtering Docs"
Cohesion: 0.25
Nodes (9): Conditional Processing via Event Filtering, highValueFilter, AnalyticsService, And (filter combinator), Filter type, FilterByMetadataExists, NewFilteredProcessor, Not (filter combinator) (+1 more)

### Community 25 - "Global Bus and Shutdown Testing"
Cohesion: 0.25
Nodes (9): Global Bus Singleton Recommendation, Synchronise Integration Tests on Shutdown, WaitForEvent Polling Helper, Global Event Bus Singleton, Read Lock Guards the Send-on-Closed-Channel Race, Idempotent Shutdown via sync.Once, Shutdown Ordering: Close, Wait, Cancel Last, Shutdown Drains the Backlog (+1 more)

### Community 26 - "Worker Pool and Global Bus"
Cohesion: 0.25
Nodes (9): GetGlobalBus, InitGlobalBus, NewEventBus, EventBus.start, EventBus.worker, Worker Pool Sizing Guidance, Workers Range Over the Channel Instead of Selecting on ctx.Done, Config (Workers, BufferSize, ShutdownTimeout) (+1 more)

### Community 27 - "Unsupported Event Type Errors"
Cohesion: 0.28
Nodes (7): UnsupportedEventTypeError, AsUnsupportedEventTypeError(), NewUnsupportedEventTypeError(), NewUnsupportedEventTypeErrorf(), TestAsUnsupportedEventTypeError(), TestIsUnsupportedEventType(), TestUnsupportedEventTypeError()

### Community 28 - "Dispatch and Graceful Degradation"
Cohesion: 0.29
Nodes (7): Dispatch and Error Collection, Graceful Degradation Philosophy, Synchronous Publishing, Copy-on-Write Unsubscribe, Worker Pool, Workers Range Over the Channel Instead of Selecting on ctx.Done, Benchmark Results and Allocation Profile

### Community 29 - "Test Data Payload Helpers"
Cohesion: 0.33
Nodes (5): LoginFailedData, LoginSuccessData, TokenRevokedData, UserCreatedData, UserUpdatedData

### Community 30 - "Event Metadata Tests"
Cohesion: 0.67
Nodes (3): T, TestEvent_WithMetadata(), TestEvent_WithMetadataOnStructLiteral()

### Community 31 - "Batch Shutdown Pitfalls"
Cohesion: 0.67
Nodes (4): Forgetting Shutdown Pitfall, Batch Processor, Batch Shutdown Flush Ordering, Batch Processing

## Ambiguous Edges - Review These
- `Go CI Build and Test Workflow` → `Gossip Examples Guide`  [AMBIGUOUS]
  .github/workflows/go.yml · relation: conceptually_related_to
- `Gossip Implementation Guide` → `GitBook Navigation Summary`  [AMBIGUOUS]
  docs/SUMMARY.md · relation: references
- `Gossip Documentation Overview` → `Auth Service Example`  [AMBIGUOUS]
  docs/README.md · relation: references
- `GitBook Navigation Summary` → `Gossip Examples Guide`  [AMBIGUOUS]
  docs/SUMMARY.md · relation: references
- `Conditional Processing via Event Filtering` → `AnalyticsService`  [AMBIGUOUS]
  docs/examples/microservices.md · relation: conceptually_related_to
- `EmailService` → `Events Treated as Immutable After Creation`  [AMBIGUOUS]
  docs/examples/microservices.md · relation: conceptually_related_to

## Knowledge Gaps
- **41 isolated node(s):** `UserCreatedData`, `LoginSuccessData`, `LoginFailedData`, `TokenRevokedData`, `UserUpdatedData` (+36 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `Go CI Build and Test Workflow` and `Gossip Examples Guide`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `Gossip Implementation Guide` and `GitBook Navigation Summary`?**
  _Edge tagged AMBIGUOUS (relation: references) - confidence is low._
- **What is the exact relationship between `Gossip Documentation Overview` and `Auth Service Example`?**
  _Edge tagged AMBIGUOUS (relation: references) - confidence is low._
- **What is the exact relationship between `GitBook Navigation Summary` and `Gossip Examples Guide`?**
  _Edge tagged AMBIGUOUS (relation: references) - confidence is low._
- **What is the exact relationship between `Conditional Processing via Event Filtering` and `AnalyticsService`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `EmailService` and `Events Treated as Immutable After Creation`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `NewEvent()` connect `Middleware Benchmarks and Auth Flow` to `Event Bus Core Dispatch`, `Batch Processor and Benchmarks`, `Event Bus Config and Tests`, `Event Filter Package`, `Safe Type Casting Utilities`, `Event Metadata Tests`?**
  _High betweenness centrality (0.212) - this node is a cross-community bridge._