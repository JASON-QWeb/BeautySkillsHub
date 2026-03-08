# Development

This document is the root entry for local development workflows.

## Unified Local Script Interface

Use the consolidated local script entrypoint:

```bash
./scripts/local.sh dev
```

Common split steps:

```bash
./scripts/local.sh db up
./scripts/local.sh migrate
./scripts/local.sh seed
```

## Recommended Reading Order

1. [README.md](./README.md)
2. [AIREAD.md](./AIREAD.md)
3. [scripts/README.md](./scripts/README.md)
4. [backend/README.md](./backend/README.md)
5. [frontend/README.md](./frontend/README.md)

## Notes

- `./scripts/local.sh` is the only public local script interface.
- Older split local helper scripts are no longer part of the repo.
- For container-first startup, prefer the instructions in [README.md](./README.md).
