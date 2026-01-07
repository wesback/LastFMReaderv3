You are implementing this repo according to:
- /speckit.specify  (behavioral requirements)
- /speckit.plan     (implementation plan, packaging, tests)

Non-negotiables:
- Go 1.22+, cobra CLI, rate limiter, backoff, NDJSON writers (local + Azure), watermark stores (file + Azure), Dockerfile (distroless), CI.
- Security: never log secrets; prefer Managed Identity for Azure.

When generating code:
- Follow package layout defined in /speckit.plan.
- Write small, testable units. Add unit tests with table-driven style.
- Respect flags/env precedence exactly as described.
- Keep logs structured (zap), redact secrets.
