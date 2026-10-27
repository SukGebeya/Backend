# CLAUDE.md

Guidance for AI coding sessions (and new contributors) working in this repository.

## What this is

Gebeya: a Go backend for an Ethiopian marketplace with escrow payments. Buyers pay through Chapa, the money is held in a Blnk ledger escrow until delivery, merchants are credited net of a 5% fee, and payouts go out through Chapa transfers. Start with `README.md`, then read:

- `docs/ARCHITECTURE.md`: how it works (layers, ledger references, order states, jobs)
- `docs/DECISIONS.md`: why it works that way (D-001 … D-017)
- `docs/ENGINEERING_LOG.md`: history of what was found and fixed, with numbers
- `docs/ROADMAP.md`: current status, launch checklist, next steps
- `docs/BUSINESS_ANALYSIS.md`: viability assessment and pilot criteria
- `docs/original-spec.md`: the original spec (historical; the code has diverged)

When you finish a meaningful piece of work, add it to `docs/ENGINEERING_LOG.md`, record any design choice in `docs/DECISIONS.md` (next number), and update `docs/ROADMAP.md`.

## Commands

```bash
make local                 # Postgres, Redis, Blnk server + worker (docker-compose-dev.yml)
make seed-blnk             # create ledger + escrow/revenue/processing-fee balances; writes IDs into .env
make run                   # API on :8080; migrations (goose + River) run on start
./scripts/e2e-money-flow.sh  # full money flow over HTTP against real Blnk, paying through mock Chapa
./scripts/e2e-endpoints.sh   # every route over HTTP; fails if any never returned 2xx
go run ./cmd/routes          # list every route (what e2e-endpoints.sh checks coverage against)
go test -short ./...       # unit tests only
go test -race ./...        # all tests; integration tests need Docker (testcontainers)
make lint                  # golangci-lint v2.15.0 (CI pins the same version)
make sqlc-generate         # after editing sqlc/queries/*.sql or migrations/
make mocks                 # after changing internal/{repository,service}/interfaces.go
make swagger               # after changing handler godoc; CI fails if docs/ is stale
```

Local Go may be newer than `go.mod`'s 1.26. If golangci-lint fails with "export data version", run it as `GOTOOLCHAIN=go1.26.0 make lint`.

## Conventions

- **Layers:** handler (parse/validate/authorize) → service (rules, one DB transaction) → repository (sqlc). No business logic in handlers. Errors go out as `internal/errors` AppErrors, and JSON goes through `pkg/response`.
- **Money:** always `int64` santim (1 ETB = 100). Use `chapa.FormatAmount` and `chapa.ParseAmountCents` for strings; never floats. Send Blnk only `precise_amount`.
- **Ledger idempotency:** every Blnk transaction uses a deterministic reference (`escrow-<order>`, `sale-credit-<order>`, `chapa-fee-<order>`, `payout-<payout>`, …). New money movements must follow the same pattern; see `internal/blnk/client.go` `createTransaction`.
- **Jobs:** enqueue with `jobs.Enqueuer.EnqueueTx` inside the same `txManager.Run` as the state change. Return `jobs.ErrNotReady` to snooze. Workers in `internal/jobs` call services through interfaces; services must not import workers.
- **Order status:** change it only through `orderRepository.Transition(id, to, from...)`.
- **Phones:** normalize with `pkg/phone` (`+2519…`/`+2517…` stored; `phone.Local` for Chapa and mobile money). Validate request fields with the `etphone` tag.
- **Queries:** sqlc reads the schema from `migrations/`. Keep SELECT column lists in table order (new columns are appended at the end), or sqlc generates per-query row types instead of the shared models.
- **Comments:** sparse, explaining *why*, matching the existing code. The lint config intentionally doesn't require doc comments.

## Testing patterns

- **Database tests:** use `testutil.NewTestDatabase(t)` and `testutil.NewTestRedis(t)`. They share one container per package, and each test gets a fresh database cloned from a migrated template. Each package with integration tests has a `TestMain` that stops the containers.
- **Service flows:** use the harness in `internal/service/harness_test.go` (`newHarness`, `seed`, `checkoutOrder`, `paidOrder`, `shippedOrder`). It runs real Postgres and mocks Blnk and Chapa (`internal/mocks`, which record call counts).
- **Development settings:** `CHAPA_MODE=mock` serves an imitation of Chapa at `/mock-chapa` (`internal/chapamock`) and points the real client at it (D-017). It has a hosted checkout (Telebirr, CBE Birr or card; pay, decline or cancel) and sends signed webhooks with a 3.5% charge in test mode. Transfers settle after `CHAPA_MOCK_TRANSFER_DELAY`, and account numbers ending in `0000` fail. Its dashboard at `http://localhost:8080/mock-chapa/` lists payments and transfers, and can reverse a payment or settle a transfer. `chapa.FakeClient` is only for unit tests. `SMS_PROVIDER=log` and `OTP_DEV_CODE=123456` are development-only; config refuses them otherwise.

## Gotchas

- Blnk enforces its API key only with `BLNK_SERVER_SECURE=true` (set in compose). The app's `BLNK_SECRET_KEY` must equal the containers' `BLNK_SERVER_SECRET_KEY`.
- Blnk 0.15 queues inflight commits and voids unless `SkipQueue: true` is sent.
- Chapa sends flat webhook payloads. `x-chapa-signature` is the HMAC of the payload; `Chapa-Signature` is the HMAC of the secret itself (static). Reversals require the former.
- `order_status` is a Postgres enum, so adding a value needs a `-- +goose NO TRANSACTION` migration.
- `carts.session_id` is UNIQUE: expired guest carts are deleted, not abandoned.
- The payout job re-checks a transfer `PAYOUT_RECHECK_INTERVAL` after starting it (snooze; 2 minutes by default, 10s in `.env.example`), so payouts complete asynchronously.
- Deleting a product or option that order items reference archives or hides it instead (`repository.ErrStillReferenced`).
