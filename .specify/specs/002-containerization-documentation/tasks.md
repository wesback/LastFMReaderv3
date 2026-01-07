# Implementation Tasks: Containerization Documentation and Configuration Management

**Feature**: 002-containerization-documentation  
**Branch**: `002-containerization-documentation`  
**Generated**: 2026-01-06  
**Updated**: Post-clarification with enhanced requirements

---

## Overview

This feature adds comprehensive documentation and configuration management for Docker and Azure Container Instances deployment. Tasks are organized by milestone, with each milestone representing an independently deliverable increment.

**Total Tasks**: 128  
**Milestones**: 6 (M1-M6)  
**Parallelizable Tasks**: 70  
**Test-First Approach**: No (documentation feature - manual validation only)

**Clarifications Applied**:
- Validation rigor: Standard (test all workflows, verify examples, check errors)
- Observability: Log Analytics, custom metrics, structured JSON logging
- Health checks: Not needed for CLI tool (removed)
- Logging format: JSON with required fields documented
- Failure scenarios: Auth, secrets, quotas, network issues

---

## Implementation Strategy

### MVP Scope
**Milestone 1 (M1)**: Configuration documentation provides immediate value for current users.

### Incremental Delivery Order
1. **M1**: Configuration Audit & Documentation (foundation for all other work)
2. **M2**: Dockerfile Enhancement (can proceed in parallel with M3)
3. **M3**: Docker Compose Environment (requires M1, independent of M2)
4. **M4**: Azure Deployment (requires M1, independent of M2/M3)
5. **M5**: Security Documentation (depends on M1, M3, M4)
6. **M6**: Testing & Validation (validates all previous milestones)

### Parallel Opportunities
- M2 and M3 can proceed in parallel after M1 completes
- M4 can proceed in parallel with M2/M3 after M1 completes
- Within each milestone, many documentation tasks are parallelizable

---

## Dependencies Graph

```
M1 (Configuration)
├─→ M2 (Dockerfile) ────┐
├─→ M3 (Docker Compose) ├─→ M5 (Security) ─→ M6 (Validation)
└─→ M4 (Azure ACI) ─────┘
```

**Blocking Dependencies**:
- M1 MUST complete before M2, M3, M4 can start
- M5 requires M1, M3, M4 to be complete
- M6 requires all previous milestones

**Independent Milestones**:
- M2, M3, M4 are independent of each other (can run in parallel)

---

## Phase 1: Setup

**Goal**: Create directory structure and prepare workspace for documentation

### Tasks

- [X] T001 Create docs/ directory in repository root
- [X] T002 Create scripts/ directory in repository root
- [X] T003 Create azure/ directory in repository root
- [X] T004 Verify existing Dockerfile location and structure
- [X] T005 Verify existing go.mod, internal/config/, and cmd/lastfm-sync/ structure

**Acceptance Criteria**:
- All required directories exist
- Existing structure verified and documented

---

## Phase 2: Foundational Tasks

**Goal**: Audit existing configuration system to establish baseline

### Tasks

- [X] T006 Audit all environment variables in internal/config/types.go
- [X] T007 Audit all CLI flags in cmd/lastfm-sync/commands/fetch.go
- [X] T008 Document configuration precedence from internal/config/loader.go
- [X] T009 Identify sensitive vs non-sensitive configuration options
- [X] T010 Extract default values from internal/config/defaults.go

**Acceptance Criteria**:
- Complete inventory of all configuration options
- Configuration precedence documented
- Sensitive fields identified

---

## Phase 3: Milestone 1 - Configuration Audit and Documentation

**User Story**: As a developer, I want complete configuration documentation so I can understand all available options and their purposes.

**Priority**: P1 (MVP)  
**Independent Test**: Developer can configure the application using only docs/configuration.md without reading source code.

### Tasks

- [X] T011 [P] [M1] Create docs/configuration.md with header and overview section
- [X] T012 [P] [M1] Add environment variables reference table to docs/configuration.md (all vars from audit)
- [X] T013 [P] [M1] Add CLI flags reference table to docs/configuration.md (all flags from audit)
- [X] T014 [P] [M1] Document configuration precedence in docs/configuration.md (flags > env > file > defaults)
- [X] T015 [P] [M1] Add validation requirements section to docs/configuration.md
- [X] T016 [P] [M1] Add examples for local development configuration in docs/configuration.md
- [X] T017 [P] [M1] Add examples for Azure configuration in docs/configuration.md
- [X] T018 [M1] Create .env.example in repository root with all variables
- [X] T019 [M1] Add comments to .env.example explaining each variable with type and purpose
- [X] T020 [M1] Mark sensitive values in .env.example with security warnings (LASTFM_API_KEY, Azure secrets)
- [X] T021 [M1] Update README.md with link to docs/configuration.md in configuration section
- [X] T022 [M1] Add configuration quick reference to README.md with most common variables

**Validation Steps**:
1. Read docs/configuration.md without looking at source code
2. Create .env file from .env.example
3. Verify all documented variables exist in code
4. Verify examples work as documented

**Deliverables**:
- docs/configuration.md (complete reference)
- .env.example (all variables with comments)
- README.md (updated configuration section)

---

## Phase 4: Milestone 2 - Dockerfile Enhancement and Documentation

**User Story**: As a developer, I want to understand the Dockerfile structure so I can build and customize the container image.

**Priority**: P2  
**Independent Test**: Developer can build container image and understand each stage's purpose by reading Dockerfile comments and docs/docker.md.

**Dependencies**: M1 (needs configuration documentation for build args)

### Tasks

- [X] T023 [P] [M2] Add stage comments to Dockerfile explaining build stage purpose and Go version
- [X] T024 [P] [M2] Add stage comments to Dockerfile explaining runtime stage and distroless benefits
- [X] T025 [P] [M2] Add comments explaining security decisions (non-root user, minimal attack surface)
- [X] T026 [P] [M2] Add comments explaining layer optimization (dependency caching, build order)
- [X] T027 [P] [M2] Create docs/docker.md with overview and architecture section
- [X] T028 [P] [M2] Add multi-stage build explanation to docs/docker.md with diagram
- [X] T029 [P] [M2] Add image size optimization section to docs/docker.md (~15-20MB target)
- [X] T030 [P] [M2] Add base image rationale to docs/docker.md (alpine vs distroless comparison)
- [X] T031 [P] [M2] Add build commands section to docs/docker.md with all options
- [X] T032 [P] [M2] Add build arguments documentation to docs/docker.md (version, build time)
- [X] T033 [P] [M2] Add caching strategies section to docs/docker.md (go mod cache optimization)
- [X] T034 [P] [M2] Add troubleshooting section to docs/docker.md (common build issues and solutions)
- [X] T035 [M2] Add Makefile targets for docker build (optional convenience commands)
- [X] T036 [M2] Update README.md with link to docs/docker.md in containerization section

**Validation Steps**:
1. Build image following docs/docker.md instructions
2. Verify build completes successfully
3. Check image size is reasonable (~15-20MB)
4. Verify non-root user with: docker run --rm lastfm-sync:dev whoami

**Deliverables**:
- Enhanced Dockerfile with inline comments
- docs/docker.md (build documentation)
- Updated README.md

---

## Phase 5: Milestone 3 - Docker Compose Development Environment

**User Story**: As a developer, I want a simple docker-compose command to start the development environment so I can test locally without complex setup.

**Priority**: P2  
**Independent Test**: Fresh repository clone → cp .env.example .env → docker compose up → successful execution.

**Dependencies**: M1 (needs .env.example)

### Tasks

- [X] T037 [M3] Create docker-compose.yml in repository root with version and services structure
- [X] T038 [M3] Add lastfm-sync service definition to docker-compose.yml
- [X] T039 [M3] Configure build context and Dockerfile reference in docker-compose.yml
- [X] T040 [M3] Add volume mounts for /data directory in docker-compose.yml
- [X] T041 [M3] Add env_file reference to .env in docker-compose.yml
- [X] T042 [M3] Add command override documentation in docker-compose.yml comments
- [X] T043 [P] [M3] Add Docker Compose section to docs/docker.md with overview
- [X] T044 [P] [M3] Document quick start workflow in docs/docker.md (clone, .env, compose up)
- [X] T045 [P] [M3] Document service structure in docs/docker.md (service name, image, volumes)
- [X] T046 [P] [M3] Document volume configuration in docs/docker.md (local persistence)
- [X] T047 [P] [M3] Add common development workflows to docs/docker.md (run, logs, cleanup)
- [X] T048 [M3] Create scripts/dev-up.sh with error checking and usage instructions
- [X] T049 [M3] Add .env validation to scripts/dev-up.sh (check required vars)
- [X] T050 [M3] Add docker availability check to scripts/dev-up.sh
- [X] T051 [M3] Create scripts/dev-down.sh for cleanup with options
- [X] T052 [M3] Add volume cleanup option to scripts/dev-down.sh (--volumes flag)
- [X] T053 [M3] Make scripts executable (chmod +x scripts/*.sh)
- [X] T054 [M3] Update .env.example with Docker Compose specific notes
- [X] T055 [M3] Update README.md with Docker Compose quick start section

**Validation Steps**:
1. Fresh clone repository
2. Run: cp .env.example .env && edit LASTFM_API_KEY
3. Run: docker compose up --build
4. Verify container starts and shows help output
5. Run: docker compose run --rm lastfm-sync fetch --user testuser
6. Verify logs show configuration loaded from .env

**Deliverables**:
- docker-compose.yml
- scripts/dev-up.sh
- scripts/dev-down.sh
- Updated docs/docker.md (Compose section)
- Updated README.md

---

## Phase 6: Milestone 4 - Azure Container Instances Documentation

**User Story**: As a DevOps engineer, I want deployment scripts and documentation so I can deploy to Azure Container Instances with proper secret management and observability.

**Priority**: P2  
**Independent Test**: Following docs/azure-deployment.md, successfully deploy to test ACI instance with secrets from Key Vault and verify logs in Log Analytics.

**Dependencies**: M1 (needs configuration documentation)

**Clarifications Applied**:
- Observability: Log Analytics, custom metrics, structured JSON logging
- Failure scenarios: Auth, secrets, quotas, network issues

### Tasks

- [X] T056 [P] [M4] Create docs/azure-deployment.md with overview and prerequisites
- [X] T057 [P] [M4] Add Azure CLI installation section to docs/azure-deployment.md
- [X] T058 [P] [M4] Add step-by-step deployment guide to docs/azure-deployment.md
- [X] T059 [P] [M4] Document resource group creation in docs/azure-deployment.md
- [X] T060 [P] [M4] Document container instance creation in docs/azure-deployment.md
- [X] T061 [P] [M4] Document environment variable configuration in docs/azure-deployment.md
- [X] T062 [P] [M4] Add Azure Key Vault integration section to docs/azure-deployment.md
- [X] T063 [P] [M4] Document managed identity setup in docs/azure-deployment.md
- [X] T064 [P] [M4] Add networking configuration section to docs/azure-deployment.md
- [X] T065 [P] [M4] Add persistent storage options to docs/azure-deployment.md
- [X] T066 [P] [M4] Add logging and monitoring section to docs/azure-deployment.md:
  - Azure Portal container logs access
  - Log Analytics workspace integration setup
  - Container exit code interpretation table
  - Custom metrics configuration (duration, API calls)
- [X] T067 [P] [M4] Add structured logging format specification to docs/azure-deployment.md:
  - JSON format requirement for Log Analytics
  - Required fields: timestamp (ISO 8601), level, message, context
  - Optional fields: user, duration_ms, api_calls, error_details
  - Example log entries
- [X] T068 [P] [M4] Add scaling considerations to docs/azure-deployment.md
- [X] T069 [P] [M4] Add cost optimization tips to docs/azure-deployment.md
- [X] T070 [M4] Create azure/deploy-aci.sh script header with usage and prerequisites
- [X] T071 [M4] Add parameter validation to azure/deploy-aci.sh (required args)
- [X] T072 [M4] Add Azure CLI availability check to azure/deploy-aci.sh
- [X] T073 [M4] Add resource group creation to azure/deploy-aci.sh
- [X] T074 [M4] Add container instance deployment to azure/deploy-aci.sh
- [X] T075 [M4] Add environment variable injection to azure/deploy-aci.sh
- [X] T076 [M4] Add secure environment variables to azure/deploy-aci.sh (Key Vault references)
- [X] T077 [M4] Add Log Analytics configuration to azure/deploy-aci.sh
- [X] T078 [M4] Add error handling and cleanup to azure/deploy-aci.sh (rollback on failure)
- [X] T079 [M4] Make azure/deploy-aci.sh executable (chmod +x)
- [X] T080 [P] [M4] Create azure/aci-params.json.example with all parameters
- [X] T081 [P] [M4] Add comments to azure/aci-params.json.example explaining each field
- [X] T082 [M4] Update README.md with Azure deployment section and link

**Validation Steps**:
1. Install Azure CLI
2. Authenticate: az login
3. Copy azure/aci-params.json.example to azure/aci-params.json
4. Edit parameters with test values
5. Run: ./azure/deploy-aci.sh
6. Verify container instance created successfully
7. Check logs: az container logs --resource-group ... --name ...
8. Verify Log Analytics integration shows logs
9. Verify structured JSON format in logs

**Deliverables**:
- docs/azure-deployment.md (with observability section)
- azure/deploy-aci.sh (executable)
- azure/aci-params.json.example
- Updated README.md

---

## Phase 7: Milestone 5 - Security and Best Practices

**User Story**: As a security-conscious user, I want documented security best practices so I can deploy safely without exposing secrets.

**Priority**: P2  
**Independent Test**: Following docs/security.md, set up Key Vault integration and verify no secrets in repository or container environment.

**Dependencies**: M1 (configuration), M3 (Docker Compose), M4 (Azure deployment)

### Tasks

- [X] T083 [P] [M5] Create docs/security.md with overview and security principles
- [X] T084 [P] [M5] Add "Never commit .env files" section to docs/security.md with examples
- [X] T085 [P] [M5] Add Azure Key Vault section to docs/security.md with setup guide
- [X] T086 [P] [M5] Add principle of least privilege section to docs/security.md (RBAC examples)
- [X] T087 [P] [M5] Add network security considerations to docs/security.md
- [X] T088 [P] [M5] Add container security best practices to docs/security.md (non-root, minimal image)
- [X] T089 [P] [M5] Add secrets management guide to docs/security.md (dev vs prod)
- [X] T090 [P] [M5] Add secret rotation strategies to docs/security.md
- [X] T091 [P] [M5] Add Azure Key Vault integration examples to docs/security.md (code snippets)
- [X] T092 [M5] Verify .gitignore contains .env entries
- [X] T093 [M5] Add .env to .gitignore if not present
- [X] T094 [M5] Add comment in .gitignore explaining .env.example exception
- [X] T095 [M5] Add security section to README.md with link to docs/security.md

**Validation Steps**:
1. Verify .env is in .gitignore: git check-ignore .env
2. Verify .env.example is tracked: git ls-files .env.example
3. Review docs/security.md for completeness
4. Follow Key Vault integration guide
5. Verify secrets not visible in container environment

**Deliverables**:
- docs/security.md
- Updated .gitignore
- Updated README.md

---

## Phase 8: Milestone 6 - Testing and Validation

**User Story**: As a new user, I want troubleshooting documentation so I can solve common issues without external help.

**Priority**: P3  
**Independent Test**: Simulate common issues (missing env vars, permission errors, Azure failures) and verify solutions in docs/troubleshooting.md work.

**Dependencies**: All previous milestones (validates everything)

**Clarifications Applied**:
- Validation rigor: Standard (test all workflows, verify examples, check errors)
- Failure scenarios: Auth, secrets, quotas, network issues with diagnostics

### Tasks

- [X] T096 [M6] Perform fresh clone test of repository on clean environment
- [X] T097 [M6] Test Docker Compose workflow from scratch (verify all steps in quickstart)
- [X] T098 [M6] Verify .env.example contains all required variables from configuration docs
- [X] T099 [M6] Test all documented docker compose commands (up, run, logs, down)
- [X] T100 [M6] Validate Dockerfile build process following docs/docker.md
- [X] T101 [M6] Test all code examples in documentation execute without errors
- [X] T102 [M6] Verify error messages are clear and actionable (test missing API key scenario)
- [X] T103 [M6] Test Azure deployment script (if Azure subscription available)
- [X] T104 [M6] Verify all configuration examples work as documented
- [X] T105 [M6] Test environment variable validation and error handling
- [X] T106 [M6] Test logging and monitoring integration (Log Analytics if available)
- [X] T107 [M6] Verify structured logging format examples are valid JSON
- [X] T108 [P] [M6] Create docs/troubleshooting.md with overview and structure
- [X] T109 [P] [M6] Add "Permission denied on data directory" to docs/troubleshooting.md with solution
- [X] T110 [P] [M6] Add "LASTFM_API_KEY is required" to docs/troubleshooting.md with validation steps
- [X] T111 [P] [M6] Add "Rate limit exceeded" to docs/troubleshooting.md with QPS adjustment
- [X] T112 [P] [M6] Add "Docker build fails" to docs/troubleshooting.md with common causes
- [X] T113 [P] [M6] Add "Container won't start" to docs/troubleshooting.md with debugging steps
- [X] T114 [P] [M6] Add Azure deployment failure scenarios to docs/troubleshooting.md:
  - Authentication failures (az login, service principal issues) with diagnostic commands
  - Missing secrets (Key Vault access, environment variables) with resolution steps
  - Resource quota errors (subscription limits, regional capacity) with workarounds
  - Network issues (connectivity, DNS resolution) with troubleshooting commands
- [X] T115 [P] [M6] Add FAQ section to docs/troubleshooting.md based on validation findings
- [X] T116 [P] [M6] Add debugging commands section to docs/troubleshooting.md with expected outputs
- [X] T117 [M6] Create validation checklist in docs/troubleshooting.md (all milestones)
- [X] T118 [M6] Update documentation based on testing findings (fix any errors discovered)
- [X] T119 [M6] Update README.md with link to docs/troubleshooting.md

**Validation Steps**:
1. Fresh clone on clean machine
2. Follow quickstart.md step-by-step
3. Intentionally trigger each documented error
4. Verify troubleshooting solutions work
5. Check all documentation links work
6. Verify all examples are accurate
7. Test Azure failure scenarios if possible

**Deliverables**:
- docs/troubleshooting.md (with Azure failure scenarios)
- Validation checklist
- Updated documentation (bug fixes from testing)
- Updated README.md

---

## Phase 9: Polish & Cross-Cutting Concerns

**Goal**: Final touches and documentation improvements

### Tasks

- [X] T120 [P] Review all documentation for consistency in tone and style
- [X] T121 [P] Verify all file paths in documentation are correct and absolute where needed
- [X] T122 [P] Check all markdown links work (internal and external)
- [X] T123 [P] Run shellcheck on all shell scripts (dev-up.sh, dev-down.sh, deploy-aci.sh)
- [X] T124 [P] Run yamllint on docker-compose.yml (if available)
- [X] T125 Verify all scripts have proper error handling and exit codes
- [X] T126 Add table of contents to long documentation files (docker.md, azure-deployment.md)
- [X] T127 Create/update CHANGELOG.md with feature details and clarifications applied
- [X] T128 Final README.md review and polish (ensure all links work)
- [X] T129 Commit all changes with descriptive commit message

**Acceptance Criteria**:
- All shellcheck warnings resolved
- All documentation links verified
- Consistent formatting and style
- Ready for merge

---

## Execution Examples

### Milestone 1 (Configuration) - Parallel Execution
```bash
# Can execute in parallel:
- T011-T017 (all docs/configuration.md sections are independent)
- T018-T020 can proceed once audit tasks (T006-T010) complete
- T021-T022 depend on T011-T017 completion
```

### Milestone 2 (Dockerfile) - Parallel Execution
```bash
# Can execute in parallel:
- T023-T026 (Dockerfile comments)
- T027-T034 (docs/docker.md sections are independent)
- T035-T036 depend on others completing
```

### Milestone 4 (Azure) - Parallel with Enhanced Observability
```bash
# Can execute in parallel:
- T056-T069 (all docs/azure-deployment.md sections independent)
  - T066: Observability section (Log Analytics, metrics)
  - T067: Structured logging format specification
- T070-T079 (deploy-aci.sh script sequential)
- T080-T081 (aci-params.json.example parallel with script)
```

---

## Task Summary

| Phase | Tasks | Parallelizable | Estimated Time |
|-------|-------|----------------|----------------|
| Setup | T001-T005 | 0 | 15 min |
| Foundational | T006-T010 | 0 | 30 min |
| M1 (Configuration) | T011-T022 | 10 | 2 hours |
| M2 (Dockerfile) | T023-T036 | 11 | 1.5 hours |
| M3 (Docker Compose) | T037-T055 | 5 | 2 hours |
| M4 (Azure) | T056-T082 | 17 | 3.5 hours |
| M5 (Security) | T083-T095 | 10 | 1.5 hours |
| M6 (Testing) | T096-T119 | 11 | 2.5 hours |
| Polish | T120-T129 | 5 | 1 hour |
| **Total** | **129** | **69 (53%)** | **~14.5 hours** |

---

## Clarifications Applied Summary

This tasks file incorporates the following clarifications from the 2026-01-06 clarification session:

1. **Validation Rigor (M6)**: Standard level - test all workflows, verify examples work, check error messages are clear
2. **Observability (M4)**: 
   - Log Analytics workspace integration
   - Container exit codes
   - Custom metrics (duration, API calls)
   - Structured JSON logging format
3. **Health Checks**: Removed - not needed for CLI tool
4. **Logging Format (M4)**:
   - JSON format with required fields: timestamp, level, message, context
   - Optional fields: user, duration_ms, api_calls, error_details
5. **Failure Scenarios (M6)**:
   - Authentication failures
   - Missing secrets
   - Resource quota errors
   - Network issues
   - All with diagnostic commands and resolution steps

---

## Notes

- **No automated tests**: This is a documentation feature. Validation is manual testing of workflows.
- **Shell script quality**: All scripts must pass shellcheck before completion.
- **Documentation validation**: All commands in documentation must be tested and work as documented.
- **Incremental value**: Each milestone delivers standalone value and can be used immediately.
- **Parallel work**: Over half the tasks can be executed in parallel within their milestone.
- **Observability focus**: M4 now includes comprehensive observability documentation per clarifications.

---

## Next Steps

1. Start with Phase 1 (Setup) - 5 quick tasks
2. Complete Phase 2 (Foundational) - establishes baseline
3. Begin Milestone 1 (Configuration) - provides immediate value
4. Proceed through remaining milestones in dependency order
5. Consider breaking work across multiple PRs per milestone for faster review
6. Pay special attention to M4 observability requirements (Log Analytics, structured logging)
7. Ensure M6 validation covers all clarified failure scenarios
