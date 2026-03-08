# Frontend Test Runner CI Fix Design

**Context**

GitHub Actions currently runs frontend lightweight tests with `node --test` directly against `.ts` files. This passes on some local machines with newer Node behavior, but fails in CI on Node 20 with `ERR_UNKNOWN_FILE_EXTENSION`. The failure is not in the tests themselves; it is in the test execution strategy and in one test file that assumes the process working directory is the repo root.

**Goal**

Make frontend lightweight tests run the same way locally and in CI, with a single supported command that works on Node 20 and does not depend on the caller's current working directory.

**Approaches**

1. Raise CI to Node 25 and keep `node --test`.
   - Smallest code change.
   - Rejected because it relies on runtime behavior drift and still leaves test execution unspecified for contributors.

2. Add an explicit TypeScript-capable runner (`tsx`) and make all frontend lightweight tests go through one npm script.
   - Stable on Node 20.
   - Makes CI and local usage identical.
   - Recommended.

3. Migrate all lightweight tests to Vitest.
   - Valid long-term option.
   - Too large for this bugfix because the current tests already fit Node’s test runner model.

**Recommended Design**

- Add `tsx` as a direct `frontend` dev dependency.
- Introduce a single frontend test entrypoint such as `npm run test:node`.
- Move the file list into one checked-in runner script under `frontend/scripts/`.
- Update `docker-runtime.test.mjs` to resolve the repository root from `import.meta.url` instead of `process.cwd()`.
- Update GitHub Actions to run `cd frontend && npm run test:node`.
- Update developer docs so the supported command is the same everywhere.
- Optionally pin local Node expectations with `.nvmrc` or `package.json.engines` so future drift is visible.

**Validation**

- `cd frontend && npm ci`
- `cd frontend && npm run test:node`
- `cd frontend && npm run build`
- `cd backend && go test ./...`
- `git diff --check`
