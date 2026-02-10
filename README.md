# compose-diff

Semantic diff for Docker Compose files — understand what actually changed.

---

## The problem

- `git diff docker-compose.yml` shows line changes, not meaning
- YAML reordering looks like massive changes
- Hard to review Compose PRs confidently
- "Did someone change the port or just reformat?"
- Missing changes in environment variables buried in noise

---

## What it does

- Parses both Compose files semantically
- Ignores formatting, ordering, whitespace
- Shows **what changed**, not **how lines moved**
- Groups changes by service
- Highlights breaking vs. non-breaking changes

---

## New in v2.0

- **Rules file support** — custom severity overrides, per-service ignores, path patterns
- **Baseline mode** — save and compare against known-good configurations
- **Category summaries** — view changes grouped by type (env, ports, images, volumes)
- **Resolved config diffing** — diff after `docker compose config` resolution

---

## Example output

```bash
$ compose-diff docker-compose.yml docker-compose.new.yml

Services:
─────────
  api:
    ⚠️  image: node:18 → node:20
    ✚  environment.NEW_VAR: "value"
    ✚  ports: 3001:3000

  postgres:
    (no changes)

  redis:
    ✖  removed service

Volumes:
────────
  ✚  added: cache_data

Networks:
─────────
  (no changes)

Summary: 2 services changed, 1 service removed, 1 volume added
```

Compare branches directly:

```bash
$ compose-diff main:docker-compose.yml feature:docker-compose.yml
```

---

## Output formats

| Format | Use case |
|--------|----------|
| `--format text` | Human-readable terminal output |
| `--format json` | CI integration, PR comments |
| `--format markdown` | GitHub PR descriptions |

---

## Change types

| Symbol | Meaning |
|--------|---------|
| `✚` | Added |
| `✖` | Removed |
| `⚠️` | Modified (potentially breaking) |
| `~` | Modified (non-breaking) |

---

## Scope

- Read-only comparison
- No file modification
- No Docker daemon required
- No telemetry

---

## Get it

**$25** — one-time purchase, standalone macOS/Linux/Windows binary.

👉 [Download on Gumroad](https://ecent.gumroad.com/l/yxzolc)

---

## Related tools

| Tool | Purpose |
|------|---------|
| **[stackgen](https://github.com/stackgen-cli/stackgen)** | Generate local dev Docker Compose stacks |
| **[envgraph](https://github.com/stackgen-cli/envgraph)** | Scan and validate environment variable usage |
| **[dataclean](https://github.com/stackgen-cli/dataclean)** | Reset local dev data safely |
| **[devcheck](https://github.com/stackgen-cli/devcheck)** | Local project readiness inspector |

---

## License

MIT — this repository contains documentation and examples only.
