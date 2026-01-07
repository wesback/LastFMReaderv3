# Feature: Containerization Documentation and Configuration Management

## Overview
Enhance LastFMReaderv3's containerization setup with comprehensive documentation for running in development and production environments (Docker Compose on Linux and Azure Container Instances), with proper configuration management through environment variables and .env files.

## Goals
- Document the existing multi-stage Dockerfile with best practices
- Provide clear guidance on required environment variables
- Create docker-compose.yml for local development
- Document Azure Container Instances deployment
- Establish secure configuration management patterns

## Clarifications

### Session 2026-01-06
- Q: What level of validation rigor should apply to Milestone 6 (Testing and Validation)? → A: Standard - Test all documented workflows + verify examples work + check error messages
- Q: What observability requirements should be documented for Azure Container Instances deployments? → A: Standard - Portal logs + Log Analytics integration + container exit codes + custom metrics + structured logging format requirements
- Q: Should health check endpoints be implemented as part of this feature or left for future work? → A: Not needed - CLI tool doesn't need health checks, remove from consideration
- Q: What should be documented regarding logging format? → A: Document recommended pattern - Specify JSON format with required fields (timestamp, level, message, context)
- Q: What failure scenarios should be documented in the Azure deployment troubleshooting section? → A: Standard - Auth failures, missing secrets, resource quota errors, network issues

## Milestones

### Milestone 1: Configuration Audit and Documentation
**Objective**: Document all configuration options and their purposes

#### Tasks
1. Audit existing environment variables and command-line flags
   - List all current configuration options
   - Document their purposes, default values, and required/optional status
   - Identify sensitive vs non-sensitive configurations
   
2. Create configuration reference documentation
   - Create `docs/configuration.md` with:
     - Complete environment variable reference table
     - Command-line flag alternatives
     - Configuration precedence order
     - Validation requirements
   
3. Create example .env file template
   - Create `.env.example` with all variables
   - Include comments explaining each variable
   - Provide sensible defaults where applicable
   - Mark sensitive values clearly

**Deliverables**:
- `docs/configuration.md`
- `.env.example`
- Updated README.md section on configuration

**Acceptance Criteria**:
- All configuration options documented
- Clear distinction between required and optional variables
- Examples provided for each variable type

---

### Milestone 2: Dockerfile Enhancement and Documentation
**Objective**: Document and optimize the existing multi-stage Dockerfile

#### Tasks
1. Review and document Dockerfile structure
   - Document each stage's purpose
   - Add inline comments explaining key decisions
   - Verify security best practices (non-root user, minimal layers)
   
2. Create Dockerfile documentation
   - Create `docs/docker.md` with:
     - Build process explanation
     - Image size optimization notes
     - Base image rationale
     - Build arguments if any
   
3. Document build process
   - Document build commands with all options
   - Explain caching strategies
   - Provide troubleshooting section

**Deliverables**:
- Enhanced Dockerfile with comments
- `docs/docker.md`
- Build scripts or Makefile targets (optional)

**Acceptance Criteria**:
- Dockerfile follows Go multi-stage build best practices
- Clear documentation of build process
- Build can be reproduced by following docs

---

### Milestone 3: Docker Compose Development Environment
**Objective**: Provide turnkey local development environment

#### Tasks
1. Create docker-compose.yml
   - Define LastFMReaderv3 service with proper configuration
   - Include volume mounts for development
   - Set up networking
   - Include any dependent services if needed
   
2. Create docker-compose documentation
   - Add section to `docs/docker.md` covering:
     - Quick start guide
     - Service descriptions
     - Volume explanations
     - Network configuration
     - Common development workflows
   
3. Create helper scripts
   - Create `scripts/dev-up.sh` for easy startup
   - Create `scripts/dev-down.sh` for cleanup
   - Include health checks

**Deliverables**:
- `docker-compose.yml`
- Updated `docs/docker.md`
- Helper scripts in `scripts/` directory
- `.env.example` updated for compose

**Acceptance Criteria**:
- `docker compose up` starts working environment
- Configuration loaded from .env file
- Logs accessible via docker compose logs
- Development workflow documented

---

### Milestone 4: Azure Container Instances Documentation
**Objective**: Enable production deployments to Azure Container Instances

#### Tasks
1. Create ACI deployment documentation
   - Create `docs/azure-deployment.md` with:
     - Prerequisites (Azure CLI, permissions)
     - Step-by-step deployment guide
     - Resource group and container creation
     - Environment variable configuration in ACI
     - Secret management with Azure Key Vault integration
   
2. Create deployment templates/scripts
   - Create `azure/deploy-aci.sh` script with:
     - Resource group creation
     - Container instance deployment
     - Environment variable injection
     - Logging configuration
   - Create `azure/aci-params.json` example
   
3. Document Azure-specific considerations
   - Networking and ingress configuration
   - Persistent storage options
   - Logging and monitoring setup:
     - Azure Portal container logs access
     - Log Analytics workspace integration
     - Container exit code interpretation
     - Custom metrics configuration (execution duration, API call counts)
     - Structured logging format requirements:
       - JSON format recommended for Log Analytics parsing
       - Required fields: timestamp (ISO 8601), level, message, context
       - Optional fields: user, duration_ms, api_calls, error_details
       - Example format specification
   - Scaling considerations
   - Cost optimization tips

**Deliverables**:
- `docs/azure-deployment.md`
- `azure/deploy-aci.sh`
- `azure/aci-params.json.example`
- Updated main README with Azure deployment link

**Acceptance Criteria**:
- Clear step-by-step ACI deployment guide
- Script successfully deploys to ACI
- Secrets management documented
- Monitoring and logging configured with:
  - Log Analytics workspace connection documented
  - Container exit codes mapped to outcomes
  - Custom metrics examples provided
  - Structured logging format specified (JSON with required fields)

---

### Milestone 5: Security and Best Practices
**Objective**: Ensure secure configuration management across environments

#### Tasks
1. Document security best practices
   - Add security section to documentation covering:
     - Never commit .env files to git
     - Use Azure Key Vault for secrets in production
     - Principle of least privilege
     - Network security considerations
   
2. Update .gitignore
   - Ensure .env files are ignored
   - Document exception for .env.example
   
3. Create secrets management guide
   - Document Azure Key Vault integration
   - Provide examples of secret retrieval
   - Document rotation strategies

**Deliverables**:
- `docs/security.md`
- Updated `.gitignore`
- Azure Key Vault integration examples

**Acceptance Criteria**:
- Security best practices documented
- .env files properly ignored
- Secrets management workflow clear

---

### Milestone 6: Testing and Validation
**Objective**: Verify all documentation and configurations work as documented

#### Tasks
1. Test local development workflow
   - Fresh clone test with docker compose
   - Verify .env.example completeness
   - Test all documented commands
   - Verify all documented workflows execute successfully
   - Validate that all examples in documentation work as written
   - Check error messages are clear and actionable
   
2. Test Azure deployment
   - Deploy to test ACI instance
   - Verify all environment variables work
   - Test logging and monitoring
   - Validate deployment script error handling
   
3. Create troubleshooting guide
   - Document common issues and solutions discovered during testing
   - Add FAQ section based on validation findings
   - Include debugging commands with expected outputs
   - Document Azure deployment failure scenarios:
     - Authentication failures (az login, service principal issues)
     - Missing secrets (Key Vault access, environment variables)
     - Resource quota errors (subscription limits, regional capacity)
     - Network issues (connectivity, DNS resolution)
     - Include diagnostic commands and resolution steps for each scenario

**Deliverables**:
- `docs/troubleshooting.md`
- Validation checklist
- Updated documentation based on testing

**Acceptance Criteria**:
- All documented workflows tested and verified working
- All code examples in documentation execute without errors
- Error messages tested and confirmed clear
- Common issues documented with validated solutions
- Fresh user can follow docs successfully without external help

---

## Documentation Structure
```
lastfmreaderv3/
├── README.md (updated with links to detailed docs)
├── Dockerfile (enhanced with comments)
├── docker-compose.yml
├── .env.example
├── .gitignore (updated)
├── docs/
│   ├── configuration.md (comprehensive config reference)
│   ├── docker.md (Docker and Compose guide)
│   ├── azure-deployment.md (ACI deployment guide)
│   ├── security.md (security best practices)
│   └── troubleshooting.md (common issues)
├── scripts/
│   ├── dev-up.sh
│   └── dev-down.sh
└── azure/
    ├── deploy-aci.sh
    └── aci-params.json.example
```

## Success Metrics
- Developer can start local environment in < 5 minutes
- Production deployment to ACI documented and scriptable
- All configuration options clearly documented
- Zero sensitive information in repository
- All common issues have documented solutions

## Dependencies
- Existing Dockerfile
- Azure subscription (for ACI testing)
- Docker and Docker Compose installed locally

## Out of Scope (for this feature)
- Kubernetes/AKS deployment (future consideration)
- CI/CD pipeline setup
- Automated testing infrastructure
- Helm charts

## Notes
- May need to add graceful shutdown handling for clean container termination
- Structured JSON logging format will be documented as recommended pattern for Azure Log Analytics integration (implementation optional)
