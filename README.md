# go-api

HTTP API template for the 2chi stack. Uses published `2chi-go-*` modules. Pairs with [`react-spa`](../react-spa) and [`nextjs-landing`](../nextjs-landing).

> **Building a new product?** Start with **[GETTING_STARTED.md](../GETTING_STARTED.md)** (fork layout, env alignment, integrations).

**Templates index:** [README.md](../README.md)

---

## Quick start (from scratch)

### 1. Shared infrastructure

From the **templates workspace root**:

```bash
./local-env/local.sh up
```

Postgres (`2chi` DB), Redis (DB 0 sessions, DB 1 rate limits), and LocalStack start on the shared stack. See [local-env/README.md](../local-env/README.md).

### 2. Environment

```bash
cd go-api
cp .env.example .env.local
```

Edit `.env.local` — secrets use the `2CHI_` prefix and override `config.yaml` (see `internals/config`). Minimum for local dev:

| Variable | Purpose |
| -------- | ------- |
| `2CHI_POSTGRES__DSN` | `postgres://postgres:postgres@localhost:5432/2chi?sslmode=disable` |
| `2CHI_REDIS__SESSION_DSN` | `redis://localhost:6379/0` |
| `2CHI_REDIS__RATE_LIMIT_DSN` | `redis://localhost:6379/1` |
| `2CHI_AUTH__ACCESS_TOKEN_SECRET` | ≥ 32 characters |
| `2CHI_AUTH__TOKEN_HASH_PEPPER` | ≥ 32 characters |

Optional integrations (needed for full billing / OAuth / address search):

| Variable | Purpose |
| -------- | ------- |
| `2CHI_GOOGLE__OAUTH__CLIENT_ID` / `CLIENT_SECRET` | Google sign-in (redirect: `http://localhost:3000/oauth/google`) |
| `2CHI_GOOGLE__MAPS__API_KEY` | Organization address autocomplete |
| `2CHI_PADDLE__API_KEY` / `WEBHOOK_SECRET` | Paddle billing |
| `2CHI_PADDLE__PRICES__*` | Must match `react-spa` `VITE_PADDLE_PRICE_*` |

VS Code launch configs load `.env.local` (see `.vscode/launch.json`).

### 3. Migrations

```bash
go run ./cmd/migrate up
```

Migrations live in `migrations/` (goose). The API also runs migrations on boot when configured.

### 4. Run API

```bash
go run ./cmd/api
```

| Service | URL |
| ------- | --- |
| API | http://localhost:1300 |
| Health | http://localhost:1300/health |
| Metrics | http://localhost:9091/metrics |

`config.yaml` sets `server.port: 1300` and CORS for `http://localhost:3000`.

### 5. Pair with SPA

```bash
cd ../react-spa
cp .env.example .env
pnpm install && pnpm dev
```

Set `VITE_API_URL=http://localhost:1300/api/v1` and align `VITE_APP_NAME` / `VITE_APP_ENV` with `config.yaml` (`app.name`, `app.environment`).

---

## Structure

```
cmd/
  api/          # HTTP server
  worker/       # SQS job consumer (apply scheduled plan changes)
  cron/         # Scheduled tasks (cleanup inline; plan changes published to SQS)
  migrate/      # goose migrations CLI
internals/
  handlers/     # HTTP layer (thin)
  services/     # Business logic
  repositories/ # Data access
  models/       # Domain types
  config/       # YAML + 2CHI_* env overrides
locales/        # Server-translated email strings (en only today)
templates/      # HTML email templates
migrations/     # SQL migrations
```

Layer order for new resources: **migration → model → repository → service → handler → gateway route**.

Product-specific domains in forks: `internals/domains/<product>/` beside `core/`.

---

## i18n and errors

| Surface | Translated? | Where |
| ------- | ----------- | ----- |
| HTTP errors | No — stable `errorCode` + optional `extra` | Services return `chi_error` codes |
| Validation | No — `errorCode: ""` + field rules in `extra` | SPA maps to `common.validation.*` |
| Transactional emails | Yes | `locales/en.json` + HTML templates |

SPA maps every handled `errorCode` in `react-spa/src/locales/en/errors.json`. Run `pnpm run check:i18n` in react-spa to guard parity.

Document new/changed codes in [`.cursor/plans/api-contracts.md`](../.cursor/plans/api-contracts.md).

---

## Paddle webhooks (local)

Forward sandbox webhooks to the local API with Hookdeck:

```bash
make hookdeck-paddle
```

Configure Paddle notification URL to the Hookdeck source URL shown on startup. Webhook path: `/api/v1/webhooks/paddle`.

Price IDs in `config.yaml` / `.env.local` must match the SPA Paddle env vars or subscription sync will fail.

---

## Makefile

| Target | Purpose |
| ------ | ------- |
| `make test` | Unit tests (`-tags=!integration`) |
| `make test-integration` | Repository integration tests (Docker) |
| `make hookdeck-paddle` | Hookdeck → local API for Paddle webhooks |

---

## Verify

```bash
make test
# or
go test ./... -count=1
```

With migration/repo changes:

```bash
make test-integration
```

---

## E2E stubs

When `react-spa` runs Playwright with `VITE_E2E=true`:

- **Location:** onboarding uses a fixed Google Place ID (`ChIJj61dQgK6j4AR4GeTYWZsY8` — 1600 Amphitheatre Parkway). Set `2CHI_GOOGLE__MAPS__API_KEY` in go-api `.env.local` so org create resolves the address.
- **Billing:** emails `e2e+*@example.com` skip live Paddle `CreateCustomer`; `ctm_e2e_*` customers are not released on rollback.

See [react-spa/e2e/README.md](../react-spa/e2e/README.md).

---

## Do not

- Return translated error messages from handlers — use `errorCode` only.
- Add SQL in handlers — use repositories.
- Fork `2chi-go-*` patterns into this repo — bump published modules instead.
