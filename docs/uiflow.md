# UI Flow

Server-rendered / embedded operator pages exposed by the API. These are
self-contained (no external CDN/JS) and served straight from the Go binary via
`go:embed`.

| Page | Route | Purpose |
| --- | --- | --- |
| Swagger UI | `GET /api/v1/docs` | Interactive OpenAPI explorer. |
| API Tester | `GET /api/v1/docs/api-test` | Phase-wise manual test workspace. |
| Deployment Status | `GET /api/v1/docs/deployments` | Live deploy pipeline dashboard. |

## Deployment Status page

Route: `GET /api/v1/docs/deployments` (public, embedded HTML). Linked prominently
from the API Tester header ("Deployment status").

Data source: the page fetches `GET /api/v1/docs/deploy-status` on load and every
30 seconds (auto-refresh). That endpoint returns:

```json
{
  "build": { "sha": "...", "sha_short": "...", "build_time": "<RFC3339>" },
  "uptime_seconds": 1234,
  "server_time": "<RFC3339>",
  "deployments": <status.json content> | null
}
```

`deployments` mirrors the pipeline-written `status.json` (bind-mounted read-only
into the api container at `DEPLOY_STATUS_FILE`). It is `null` in local dev or when
the file is unset/missing/invalid.

### What it shows

- **Running version** — `sha_short` + build time of the binary currently serving.
- **Uptime** — process uptime derived from `uptime_seconds`.
- **Freshness badge** — compares `build.sha` to `deployments.latest_main.sha`:
  - green **Up to date** when they match,
  - amber **Behind main** when origin/main is ahead,
  - red **HELD — last deploy failed** when `deployments.held` is non-null,
  - neutral **No pipeline data** when `deployments` is `null`.
- **Held banner** — shown when `held` is set (reason, held SHA, failure count,
  since). Explains that a new `origin/main` commit clears the hold.
- **Deploy attempts** — newest-first cards, outcome color-coded (green `success`,
  red `failed`, amber `rolled_back`). Each card shows the commit sha_short and
  subject, trigger (`auto`/`manual`), duration, per-phase chips
  (`prechecks → build → backup → migrate → swap → postchecks → revert`, each
  `ok`/`failed`/`skipped` with duration), and an expandable `<details>` block for
  the sanitized `error_tail` when present.

### Empty / error states

- `deployments == null` → "No deployment data" placeholder (build identity still
  shown). This is the normal local-dev state.
- No attempts recorded → "No attempts recorded yet".
- `deploy-status` unreachable → "Unable to load status" with the fetch error.
