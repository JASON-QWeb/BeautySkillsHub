# Docker Compose And CI Alignment Design

**Context**

The frontend container was recently hardened to run as non-root, but `docker compose up -d --build` exposed a runtime mismatch in the nginx image choice. A second review of the local Docker stack shows another fragility: the compose backend service still depends on an ignored `backend/.env.local`, which means a fresh clone or CI-style environment cannot start the stack without a private local file.

**Goal**

Make the Docker configuration production-grade and still frictionless for local development. A fresh local checkout should be able to build and boot the full stack with `docker compose up -d --build` and without any extra manual configuration.

**Approaches**

1. Keep `env_file` and document the requirement.
   - Smallest change.
   - Rejected because it preserves the current fresh-clone failure mode and leaks local secrets through compose inspection.

2. Replace backend compose secrets with safe local defaults in `docker-compose.yml`, keep image/runtime hardening, and teach CI to verify Docker artifacts.
   - Keeps local startup zero-config.
   - Avoids private-file dependency.
   - Lets CI catch future Docker regressions.
   - Recommended.

3. Split local and CI compose files.
   - Flexible but adds maintenance overhead and more chances for drift.
   - Not necessary for the current stack.

**Recommended Design**

- Keep the backend and frontend containers non-root.
- Use `nginxinc/nginx-unprivileged:alpine` for the frontend runtime image.
- Remove `backend/.env.local` from `docker-compose.yml` and replace it with explicit safe local defaults:
  - `APP_ENV=local`
  - local `DATABASE_URL`
  - stable local `JWT_SECRET`
  - empty `OPENAI_API_KEY`
  - `GITHUB_SYNC_ENABLED=false`
  - local Redis defaults
- Keep all values overrideable through shell environment variables.
- Update Docker regression tests to assert:
  - frontend uses the unprivileged nginx image
  - backend runs as non-root
  - compose does not depend on `backend/.env.local`
  - compose injects safe local defaults
  - CI verifies Docker images and compose boot
- Update `.github/workflows/verify.yml` so GitHub Actions also builds the Docker images and performs a compose smoke test.

**Validation**

- `node --test frontend/docker-runtime.test.mjs`
- `docker build -f backend/Dockerfile backend`
- `docker build -f frontend/Dockerfile frontend`
- `docker compose up -d --build`
- `docker compose ps`
- `curl -I http://127.0.0.1:5173`
- `curl -sf http://127.0.0.1:8080/health`
- `cd backend && go test ./...`
- `cd frontend && npm run build`
