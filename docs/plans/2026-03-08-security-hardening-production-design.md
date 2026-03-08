# Security Hardening Production Design

## Goal

Harden the backend request pipeline to production standards by replacing wildcard CORS with an explicit allowlist, adding consistent security response headers, and enforcing route-specific rate limiting for authentication and AI-cost-sensitive endpoints. The work also needs a review-fix document aligned with the 2026-03-08 architecture review.

## Scope

This design covers three review findings from [docs/review/2026-03-08-architecture-review.md](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/docs/review/2026-03-08-architecture-review.md):

- CORS uses `*` instead of trusted origins.
- Responses are missing baseline security headers.
- Authentication and AI-triggering endpoints have no request rate limiting.

Secret rotation and git history rewriting are intentionally out of scope for this change because the current request is limited to the three middleware concerns above.

## Approach Options

### Option A: App-layer hardening with config-driven middleware

Build first-party middleware for CORS, security headers, and rate limiting, all configured from backend environment variables and wired in the Gin server.

Pros:

- Fully testable inside this repository.
- Works the same in local, staging, and production.
- No dependency on ingress or CDN policy for core safety.

Cons:

- Adds some server-side configuration surface area.
- Requires careful defaults to avoid breaking local development.

### Option B: Split responsibility between app and ingress

Keep minimal middleware in the app and expect reverse proxy / ingress to provide security headers and most rate limiting.

Pros:

- Smaller application diff.
- Common in large deployments with centralized edge policies.

Cons:

- Not self-contained or testable in this repo.
- Easy for environments to drift.
- Review finding remains only partially fixed in code.

### Option C: Third-party middleware packages

Adopt external Gin middleware packages for security headers and rate limiting.

Pros:

- Faster initial implementation.

Cons:

- Adds new dependencies without solving project-specific route policy decisions.
- Harder to reason about fallback behavior and tests.

## Recommendation

Use Option A. It produces the most defensible repository-level fix and can still coexist with stricter ingress controls later.

## Detailed Design

### Configuration

Add explicit backend configuration for:

- Allowed CORS origins as a comma-separated list.
- Optional allowed CORS headers and exposed headers with safe defaults.
- CSP policy string and a `report-only` switch.
- HSTS enablement and max age.
- Rate-limit backend selection that prefers Redis and falls back to in-memory if Redis is not configured.
- Per-route limiter windows and budgets for login, registration, review retry, and AI chat.
- Trusted proxy CIDRs / forwarded-IP trust toggle only if needed by Gin defaults.

Local development defaults should include localhost origins. Production behavior should default closed if no allowlist is configured.

### CORS

Replace the current wildcard CORS middleware with an origin-matching middleware that:

- Allows only explicitly configured origins.
- Reflects the request origin when it matches.
- Sets `Vary: Origin` and preflight-specific `Vary` headers.
- Handles `OPTIONS` preflight consistently.
- Rejects disallowed cross-origin requests with `403`.
- Leaves requests without an `Origin` header untouched so non-browser clients continue to work.

### Security Headers

Add a global middleware that sets:

- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy` with a deny-by-default baseline
- `Cross-Origin-Opener-Policy: same-origin`
- `Cross-Origin-Resource-Policy: same-site`
- `Content-Security-Policy` or `Content-Security-Policy-Report-Only` from config
- `Strict-Transport-Security` only when enabled by config and the request is HTTPS / forwarded as HTTPS

Do not add deprecated `X-XSS-Protection`. Modern browsers ignore it and it creates noise without value.

### Rate Limiting

Add a reusable limiter abstraction with two implementations:

- Redis-backed sliding window / token bucket for production and multi-instance safety.
- In-memory fallback for local and single-instance execution.

Apply it to:

- `POST /api/auth/login`
- `POST /api/auth/register`
- `POST /api/skills/:id/review/retry`
- `POST /api/rules/:id/review/retry`
- `POST /api/ai/chat`

Key strategy:

- Authentication routes: client IP + route.
- Review retry: authenticated user ID + route, fallback to IP if auth context is missing.
- AI chat: authenticated user ID if present, else IP.

Rate-limit responses should return `429` JSON with `Retry-After`.

### Testing

Follow TDD:

- Add failing middleware tests for CORS allow/deny behavior.
- Add failing middleware tests for security header presence and HSTS gating.
- Add failing rate-limit tests for in-memory limiter behavior and JSON `429`.
- Add server wiring tests that prove guarded routes enforce the limiter while ordinary routes do not.

### Documentation

Create a matching fix record under `docs/review_fix/2026-03-08-architecture-review.md` summarizing:

- Findings addressed
- Root cause
- Code changes
- Config knobs
- Verification commands
- Remaining operational recommendations

## Risks

- Misconfigured origin allowlists can block the frontend immediately.
- Overly strict CSP can break the UI if asset sources are not accounted for.
- Rate limits can block legitimate traffic if budgets are too low or proxy IP handling is wrong.

## Mitigations

- Ship safe local defaults and explicit production variables.
- Keep CSP configurable instead of hard-coding a brittle string.
- Prefer authenticated identity over IP when available.
- Use Redis automatically when configured to avoid per-instance drift.
