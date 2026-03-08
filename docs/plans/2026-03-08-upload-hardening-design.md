# Upload Hardening Design

**Date:** 2026-03-08

**Context**

The backend currently enforces upload size limits inside handlers after `FormFile` / `MultipartForm` parsing. That leaves multipart parsing exposed to oversized request bodies and allows unnecessary memory, disk, and temp file pressure before the request is rejected. Thumbnail storage also uses deterministic names derived only from the resource name, which allows same-name uploads to overwrite or delete each other's thumbnails. The repository additionally tracks build artifacts that should remain local.

**Goals**

- Reject oversized multipart requests before handler-level parsing.
- Preserve existing business limits for files, thumbnails, and content images.
- Prevent thumbnail collisions across same-name uploads.
- Tighten ignore rules so local build/cache artifacts stay out of git.

**Approach**

1. Add a multipart-limiting Gin middleware that wraps the request body with `http.MaxBytesReader` before parsing and eagerly parses multipart requests. The middleware will:
   - enforce route-specific hard request caps,
   - return `413` for oversized requests before upload handlers run,
   - clean up `MultipartForm` temp files after the request finishes.
2. Apply the middleware only to multipart-capable upload/update routes:
   - reviewed skill/rules upload,
   - MCP/tools/rules resource create and update,
   - content asset image upload.
3. Change stored thumbnail naming from `slug + "_thumb" + ext` to `slug + "-" + random suffix + "_thumb" + ext`.
4. Expand root `.gitignore` coverage for backend binaries, avatars, TypeScript build info, SQLite variants, and temp files, then stop tracking cache/binary artifacts already committed.

**Rejected Alternatives**

- Global upload cap on all routes: too coarse and weak for smaller endpoints like content images.
- Hash-based thumbnail deduplication: adds shared-lifecycle complexity without solving the immediate collision bug more simply than random suffixes.
- Relying only on reverse proxy limits: insufficient for direct backend access and local/dev environments.

**Testing**

- Middleware unit tests for oversized multipart rejection and normal pass-through.
- Thumbnail storage tests proving same-name uploads generate distinct stored filenames.
- Existing backend test suite and frontend build after changes.
