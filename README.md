# compose-diff

**Free and open source** — Semantic diff for Docker Compose files. See what actually changed, not just which lines moved.

Stop squinting at YAML diffs. compose-diff tells you in plain English: which services changed, what environment variables were added or removed, which ports shifted, and what might break.

## The Problem

```diff
- DATABASE_URL=postgres://...
+ DB_URL=postgres://...
```

A line-based diff shows text changed. compose-diff shows:
- ⚠️ `services.api.environment.DATABASE_URL` **removed** (PotentialBreaking)
- ➕ `services.api.environment.DB_URL` **added**

## Features

- **Semantic comparison** — understands services, ports, volumes, env vars, networks
- **Breaking change detection** — flags removed ports, deleted env vars, image changes
- **Rules file support** — custom severity overrides, per-service ignores, path patterns
- **Baseline mode** — save and compare against known-good configurations
- **Category summaries** — view changes grouped by type (env, ports, images, volumes)
- **Resolved config diffing** — diff after `docker compose config` resolution
- **Multiple outputs** — text, JSON, or Markdown for PR comments
- **Deterministic** — same inputs always produce same outputs
- **Offline** — single binary, no network required

## Quick Start

```bash
# Compare two compose files
compose-diff diff docker-compose.old.yml docker-compose.new.yml

# JSON output for CI
compose-diff diff --format json old.yml new.yml

# Focus on one service
compose-diff diff --service api old.yml new.yml

# Fail CI if breaking changes detected
compose-diff diff --strict old.yml new.yml

# Use custom rules file
compose-diff diff --rules diff-rules.yaml old.yml new.yml

# Save current as baseline
compose-diff diff --save-baseline baseline.json docker-compose.yml

# Compare against baseline
compose-diff diff --baseline baseline.json docker-compose.yml

# Show category summary
compose-diff diff --category old.yml new.yml

# Diff resolved configs (after variable substitution)
compose-diff diff --resolve old.yml new.yml
```

## Rules File

Create a rules file to customize severity and ignores:

```yaml
# Severity overrides for specific paths
severity_overrides:
  environment.DEBUG: info
  environment.LOG_LEVEL: info
  image: warning

# Per-service rules
services:
  api:
    ignore_paths:
      - environment.DEV_*
    severity_overrides:
      ports: breaking
  
# Global ignores (regex patterns)
global_ignores:
  - ".*_TEST_.*"
  - "environment.LOCAL_.*"
```

## Example Output

```
compose-diff v1.0.0

Comparing: docker-compose.old.yml → docker-compose.new.yml

Summary: 2 services changed, 1 added, 0 removed
         5 changes (1 breaking, 2 warnings, 2 info)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Service: api
  ⚠️  BREAKING  environment.DATABASE_URL removed
  ⚡ WARNING   image changed: node:18 → node:20
  ➕ ADDED     environment.DB_URL = "postgres://..."

Service: redis
  ➕ ADDED     ports: 6379:6379

Service: worker (NEW)
  ➕ ADDED     image: myapp/worker:latest
```

## What It Is / What It Isn't

**It is:**
- A diff tool for Docker Compose configurations
- A way to catch config drift between branches
- Useful for PR reviews and change documentation

**It is not:**
- A validator (use `docker compose config`)
- A deployment tool
- A security scanner
- Production-grade change management

## Installation

Download the binary for your platform from [Gumroad](https://example.gumroad.com/l/compose-diff).

```bash
# macOS / Linux
chmod +x compose-diff
sudo mv compose-diff /usr/local/bin/

# Verify
compose-diff version
```

## Flags Reference

| Flag | Description |
|------|-------------|
| `--format` | Output format: `text`, `json`, `markdown` |
| `--service` | Filter to specific service |
| `--severity` | Minimum severity: `info`, `warning`, `breaking` |
| `--strict` | Exit 1 if breaking changes detected |
| `--color` | Color output: `auto`, `always`, `never` |
| `--normalize` | Normalize before diff (default: on) |
| `--rules` | Custom rules file for severity overrides |
| `--baseline` | Compare against baseline file |
| `--save-baseline` | Save current state as baseline |
| `--category` | Show category summary (env, ports, images, volumes) |
| `--category-detail` | Show detailed category breakdown |
| `--resolve` | Run `docker compose config` before diffing |

## Exit Codes

- `0` — Diff completed successfully
- `1` — Diff completed, `--strict` mode and breaking changes found
- `2` — Parse error or invalid input

## JSON Schema

```json
{
  "schema_version": "1.0",
  "summary": {
    "services_added": 1,
    "services_removed": 0,
    "services_changed": 2,
    "total_changes": 5,
    "breaking_count": 1
  },
  "changes": [
    {
      "kind": "removed",
      "scope": "service",
      "name": "api",
      "path": "services.api.environment.DATABASE_URL",
      "before": "postgres://...",
      "after": null,
      "severity": "breaking"
    }
  ]
}
```

## Related Tools

compose-diff is part of a local development toolchain:

- **[stackgen](https://github.com/ecent1119/stackgen)** — Generate docker-compose.yml for any stack
- **[envgraph](https://github.com/ecent1119/envgraph)** — Visualize environment variable flow
- **[dataclean](https://github.com/ecent1119/dataclean)** — Snapshot and reset Docker volumes
- **[devcheck](https://github.com/ecent1119/devcheck)** — Verify local dev prerequisites

## Support This Project

**compose-diff is free and open source.**

If this tool saved you time, consider sponsoring:

[![Sponsor on GitHub](https://img.shields.io/badge/Sponsor-❤️-red?logo=github)](https://github.com/sponsors/ecent1119)

Your support helps maintain and improve this tool.

## License

MIT License — see [LICENSE](LICENSE) for details.
