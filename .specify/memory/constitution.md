<!-- 
Sync Impact Report
==================
Version Change: 1.0.0 → 1.0.0 (Initial Constitution)
New Principles: Test-First Development, Code Quality Standards, User Experience Consistency, Performance Requirements
New Sections: Code Quality Standards, Performance Requirements, UX Consistency Standards, Development Workflow
Templates Updated: ✅ spec-template.md (UX requirements), ✅ tasks-template.md (testing/performance tasks)
Follow-up: None
-->

# LastFM Reader v3 Constitution

## Core Principles

### I. Test-First Development (NON-NEGOTIABLE)

All features MUST follow Test-Driven Development (TDD) discipline:
- Unit tests MUST be written before implementation code
- Red-Green-Refactor cycle is mandatory: red (failing test) → green (implementation) → refactor (optimization)
- Test specifications MUST be user-approved before implementation begins
- MINIMUM test coverage: 80% for new code, 75% overall
- Every user story MUST have at least one independent integration test demonstrating end-to-end functionality
- Tests MUST be maintainable and serve as living documentation of expected behavior

Rationale: Test-first ensures requirements clarity, prevents regression, and provides confidence for future refactoring.

### II. Code Quality Standards (NON-NEGOTIABLE)

Every committed code change MUST adhere to documented code quality standards:
- Consistent linting rules enforced across all source files (TypeScript/JavaScript ESLint or equivalent)
- Complexity limits: Cyclomatic complexity < 10 per function, cognitive complexity < 15
- Code reviews MUST verify adherence before merge; non-compliance blocks PR approval
- MUST use type-safe languages/practices: TypeScript preferred, strict null checking enforced
- Code duplication: DRY principle enforced; identical code blocks > 3 lines MUST be refactored into shared utilities
- Comments MUST explain WHY, not WHAT; code MUST be self-documenting via clear naming
- Automated linting and formatting tools MUST run pre-commit (via git hooks)

Rationale: Consistent quality reduces maintenance burden, improves debuggability, and enables confident refactoring.

### III. User Experience Consistency

All user-facing features MUST deliver consistent, predictable interactions:
- User workflows MUST follow established UX patterns (see UX Consistency Standards section)
- Error messages MUST be clear, actionable, and follow the same format/tone
- Data presentation MUST use consistent units, formats, and terminology across all interfaces
- Loading states, empty states, and error states MUST be explicitly designed for every feature
- Accessibility standards (WCAG 2.1 Level AA minimum) MUST be met for all UI components
- User testing or feedback validation MUST occur before feature release

Rationale: Consistency builds user confidence, reduces learning curve, and enables accessible experiences.

### IV. Performance Requirements

Performance characteristics MUST be measured and meet established targets:
- MUST define performance budgets for critical user journeys (e.g., page load < 2s, API response < 500ms)
- Monitoring/metrics collection MUST be built-in; performance regressions MUST trigger alerts
- Code changes impacting performance MUST include before/after metrics in PR description
- Database queries MUST be optimized: N+1 queries prohibited, indexes required for > 10K records
- Client-side bundles MUST be < 250KB (gzipped) for initial load; lazy loading for features > 100KB
- Memory leaks MUST be prevented: cleanup/disposal patterns required in event listeners, subscriptions, and timers

Rationale: Predictable performance ensures reliability, user satisfaction, and cost efficiency at scale.

### V. Independent User Story Testing

Each user story MUST be independently testable and deployable:
- User stories are prioritized as slices (P1, P2, P3) that each deliver distinct value when implemented in isolation
- Every story MUST have an "Independent Test" definition describing how it validates independently
- P1 stories MUST be tested and deployable before P2 stories begin
- Features MUST NOT block each other; shared infrastructure implemented first

Rationale: Independent testing enables MVP increments, reduces risk, and allows parallel work streams.

## Code Quality Standards

### Linting & Formatting

- TypeScript projects use ESLint with recommended TypeScript rules
- Prettier is mandatory for code formatting; all code must pass `prettier --check`
- Pre-commit git hooks run linting automatically (must not be bypassed)
- CI/CD pipeline fails on linting violations; no exceptions

### Complexity Thresholds

| Metric | Threshold | Measurement |
|--------|-----------|-------------|
| Cyclomatic Complexity | < 10 | per function |
| Cognitive Complexity | < 15 | per function |
| Lines per Function | < 50 | average; < 100 max |
| Function Parameters | ≤ 4 | positional args |

### Type Safety

- Strict null checks enabled (`noImplicitAny: true, strictNullChecks: true`)
- All function parameters and return types explicitly annotated in TypeScript
- No `any` types except in legacy/third-party integrations (document with `// @ts-ignore` reason)
- Interfaces/types used for all object structures; avoid loose object types

## Performance Requirements

### Performance Budgets

| Metric | Target | Environment |
|--------|--------|-------------|
| Initial Page Load | < 2 seconds | 4G network, mid-range device |
| API Response Time | < 500ms p95 | typical query |
| Bundle Size (gzipped) | < 250KB | initial load |
| Lazy-loaded Features | < 100KB | each module |
| Time to Interactive | < 3 seconds | 4G network |

### Monitoring & Measurement

- Performance metrics collected and reported weekly
- Regressions automatically flagged if > 10% slower than baseline
- Long-running operations (> 1s) must show progress indicators or background tasks
- Database queries logged with execution time; queries > 1s reviewed for optimization

### Optimization Practices

- API calls batch requested where possible (avoid N+1 query patterns)
- Database indexes created for columns queried in WHERE/JOIN conditions (> 10K records)
- Lazy loading implemented for features > 100KB
- Client state management limits in-memory caches to < 50MB
- Memory cleanup required: event listeners removed, subscriptions unsubscribed, timers cleared

## UX Consistency Standards

### User Interaction Patterns

- Loading states: Spinner or skeleton + message explaining what's loading
- Empty states: Illustrated message + action to populate content (e.g., "Import your music")
- Error states: Clear message (what failed), reason (why), action (how to fix)
- Success feedback: Toast notification or banner for < 3s operations; inline message for longer tasks
- Form validation: Real-time feedback for each field; summary of errors before submission

### Data Presentation

- Dates always in ISO 8601 format (YYYY-MM-DD) in UI; user locale respected for display
- Numbers formatted consistently (decimals, thousands separators, units)
- Units always explicit: "Duration: 3m 45s", not "Duration: 225" 
- Terminology consistent across all screens (e.g., "playlist" not sometimes "playlist", sometimes "collection")
- Color usage accessible: no information conveyed by color alone; red/green colorblind safe

### Accessibility

- WCAG 2.1 Level AA compliance mandatory for all user interfaces
- Screen reader testing performed on new features before release
- Keyboard navigation fully functional (Tab, Enter, Escape, arrow keys)
- Focus indicators visible; focus management implemented for modals and dynamic content

## Development Workflow

### Code Review Process

1. All code changes require review by ≥ 1 peer before merge
2. Reviewer MUST verify: (a) tests pass, (b) code quality standards met, (c) no performance regressions
3. Test coverage must not decrease; new features require 80% coverage
4. Code comments explaining complex logic or non-obvious decisions required
5. Breaking changes flagged and documented

### Testing Gate

- Unit tests MUST pass locally before push
- CI/CD pipeline runs all tests; PR blocked if any fail
- Integration tests covering user workflows (minimum: 1 per user story) MUST pass
- Performance tests (e.g., bundle size, load time) MUST not regress > 10%

### Performance Review

- Performance metrics included in PR description for changes impacting: database queries, bundle size, or response times
- Existing performance tests must pass; new performance-critical features require new benchmarks
- Production metrics reviewed weekly; spikes investigated within 24 hours

### Deployment Approval

- Code review ✅
- Tests ✅ (unit + integration + performance)
- Performance metrics ✅ (no unexplained regressions)
- Documentation ✅ (code comments, user guides if applicable)

## Governance

### Constitution Authority

This constitution supersedes all informal practices and guidelines. All PRs and reviews MUST verify compliance with these principles. Exceptions require explicit documentation (comment in PR with rationale approved by ≥ 2 reviewers).

### Amendment Process

1. **Proposal**: Documented in issue with rationale and impact assessment
2. **Discussion**: Team feedback period (minimum 1 week)
3. **Ratification**: Requires unanimous consensus or 75% majority vote with documented dissent
4. **Documentation**: Amendment PR includes updated constitution, version bump, and template updates
5. **Migration**: Timeline and guidance for existing code to comply

### Versioning

- MAJOR: Backward incompatible governance changes (e.g., principle removal, threshold reduction)
- MINOR: New principles, new sections, expanded standards
- PATCH: Clarifications, wording updates, non-semantic refinements
- All versions recorded with ISO 8601 date and author

### Compliance Monitoring

- Automated: Linting, test coverage, performance metrics flagged by CI/CD
- Manual: Code reviews verify principle adherence; feedback loop surfaces violations
- Quarterly: Team retrospective reviews constitution effectiveness; improvement proposals documented

**Version**: 1.0.0 | **Ratified**: 2025-10-30 | **Last Amended**: 2025-10-30

```
