# GoFast Project

## Overview

This project provides a full-stack foundation built with:

- Go 1.25 service (`service-core`) using ConnectRPC and Goose-backed Postgres migrations
- SvelteKit client (`service-client`) built with Vite
- PostgreSQL 18 database
- Local observability stack (Alloy, Prometheus, Tempo, Loki, Grafana) wired through OTLP

All compute and client containers are orchestrated with Docker Compose, and the `gof` CLI keeps code, queries, and client pages in sync.

## Prerequisites

- Docker Engine with Compose v2 (`docker compose`)
- [Mise](https://mise.jdx.dev/) for managing Go, Node, and other tools.
- `sqlc` (see [installation guide](https://docs.sqlc.dev/en/latest/overview/install.html))
- `goose` (see [installation guide](https://github.com/pressly/goose#install))

This project uses `mise` to manage versions for Go, Node, buf, and golangci-lint. Install `mise`, then run `mise install` in the project root.

`sqlc` and `goose` must be installed separately.

Authenticate once with GoFast before running CLI generators:

```bash
gof auth
```

## Quick Start

1. Copy `.env.example` to `.env` (if not present) and adjust secrets as needed.
2. Launch the local environment:

   ```bash
   sh start.sh
   ```

3. Visit:
   - Client UI: http://localhost:3000
   - Core API: http://localhost:4000
   - Grafana: http://localhost:3001 (anonymous admin login)

Stop everything with:

```bash
sh stop.sh
```

## Helpful Scripts

| Script | Purpose |
| --- | --- |
| `scripts/run_keys.sh` | Regenerate JWT signing keys |
| `scripts/run_proto.sh` | Regenerate Buf protobuf code |
| `scripts/run_queries.sh` | Regenerate sqlc query code |
| `scripts/run_migrations.sh` | Apply Goose migrations against local Postgres |
| `scripts/run_tests.sh` | Run Go unit tests + SvelteKit integration tests + E2E tests |
| `scripts/seed_dev_user.sh` | Seed a development user for quick login |

## GoFast CLI Commands

| Command | Description |
| --- | --- |
| `gof auth` | Store GoFast credentials for future CLI calls |
| `gof init <project>` | Scaffold a new GoFast project into `<project>` |
| `gof client svelte` | Add the SvelteKit client and docker-compose overrides |
| `gof model <name> field:type ...` | Generate DB migrations, service, transport, and Svelte views |
| `gof infra` | Copy observability/infra templates and mark the project as configured |
| `gof version` | Print CLI version information |

Use `gof <command> --help` for additional flags and examples.

## Project Layout

- `app/service-core`: Go service, generated queries, migrations, and transport layer
- `app/service-client`: SvelteKit client wired to ConnectRPC APIs
- `scripts/`: Automation for code generation, migrations, tests, and seeding
- `infra/`: Terraform, Kubernetes, and cloud bootstrap scripts (see `infra/README.md`)
- `monitoring/`: Configuration for Alloy, Prometheus, Tempo, Loki, and Grafana
- `start.sh`: Convenience script to run core, client, and observability compose files together
- `stop.sh`: Convenience script to stop all running containers

## Next Steps

Ready to move past local development? Follow `infra/README.md` for provisioning Kubernetes (RKE2), GitHub secrets, GCP resources, and Cloudflare Workers deployment workflows. Each section links to the corresponding `setup_*.sh` script for a guided, scriptable rollout.
