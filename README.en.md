<div align="center">

# BeautySkillsHub

### AI Resource Aggregation Platform Template

English | [简体中文](./README.md)

</div>

<p align="center">
  <img src="./demo1.png" alt="BeautySkillsHub demo screenshot 1" width="49%" />
  <img src="./demo2.png" alt="BeautySkillsHub demo screenshot 2" width="49%" />
</p>

<p align="center">
  A ready-to-use AI resource aggregation platform template with built-in Skill upload to GitHub,
  AI review workflows, and AI assistant capabilities.
  <br />
  It helps you quickly build a shared AI platform for aggregating <code>skill</code>, <code>rules</code>,
  <code>mcp</code>, <code>tools</code>, and other resource types for both B2B and consumer-facing scenarios.
</p>

<p align="center">
  <a href="./AIREAD.md"><strong>AI Quick Start</strong></a>
  ·
  <a href="#quick-start"><strong>Quick Start</strong></a>
  ·
  <a href="#overview"><strong>Overview</strong></a>
  ·
  <a href="#repository-guide"><strong>Repository Guide</strong></a>
</p>

## Recommended: Let AI Read and Boot the Project

Your AI coding assistant can start with [AIREAD.md](./AIREAD.md) to understand the project layout, runtime flow, and main development entry points.

## Quick Start

The recommended way is to boot the full local stack directly:

```bash
docker compose up -d --build
```

Default endpoints after startup:

- Frontend: `http://localhost:5173`
- Backend health check: `http://localhost:8080/health`
- Backend API: `http://localhost:8080/api/...`

To stop the stack:

```bash
docker compose down
```

If you prefer to run frontend and backend directly on the host machine:

```bash
./scripts/local.sh dev
```

## Overview

BeautySkillsHub is a decoupled full-stack resource platform built around the workflow of upload, review, publish, discovery, and reuse. It already includes these core capabilities:

- `skill / rules` follow an AI review + human review + revision update workflow
- `mcp / tools` follow an auto-approve + auto-publish workflow
- Built-in likes, favorites, download statistics, and paginated personal uploads
- PostgreSQL migration-first schema management
- Built-in `/health`, secure response headers, CORS allowlist, rate limiting, and non-root container runtime

Tech stack:

- Backend: Go + Gin + GORM
- Frontend: React + Vite + TypeScript
- Data layer: PostgreSQL + Redis
- Runtime: Docker Compose or direct host-machine development

## Repository Guide

- [AIREAD.md](./AIREAD.md): shortest onboarding path for AI coding assistants
- [backend/README.md](./backend/README.md): backend structure, startup, and testing
- [frontend/README.md](./frontend/README.md): frontend structure, build, and testing
- [scripts/README.md](./scripts/README.md): local script entrypoints
- [db/SCHEMA.md](./db/SCHEMA.md): database schema and migration notes
