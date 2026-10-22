# Gebeya: escrow marketplace backend for Ethiopia

[![CI](https://github.com/SukGebeya/Backend/actions/workflows/ci.yml/badge.svg)](https://github.com/SukGebeya/Backend/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)
![Postgres](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**Gebeya** (ገበያ, "marketplace") is a backend for small and informal merchants in Ethiopia. Merchants get a storefront and a wallet. Buyers pay with Telebirr, CBE Birr or card through [Chapa](https://chapa.co), and the money is held in escrow until the buyer confirms delivery. Every birr is recorded in a [Blnk](https://github.com/blnkfinance/blnk) double-entry ledger, which becomes each merchant's verifiable sales history: the basis for working-capital loans later.

**Frontend:** a SvelteKit web app for buyers and sellers lives in [SukGebeya/Frontend](https://github.com/SukGebeya/Frontend).

## Highlights

- **Escrow that reconciles.** Payments are re-verified with Chapa (status, exact amount, live mode) before escrow is created. Settlement, refunds, chargebacks and payouts are retryable background jobs, idempotent through deterministic ledger references. A daily job cross-checks the ledger against Postgres.
- **Built for the Ethiopian market.**
  - Phone-first accounts: `0911…`, `+251 91…` and `911…` are the same user, verified by SMS in Amharic and English.
  - Chapa's field rules are handled.
  - Addresses use subcity and woreda.
  - Search works in Ge'ez script.
  - Merchant delivery fees, disputes, and auto-confirmed delivery.
  - KYC-gated payouts to Telebirr, CBE Birr and banks.
- **Correct under failure and concurrency.**
  - Jobs are enqueued in the same database transaction as the change that needs them.
  - Order status changes are compare-and-set.
  - Stock reservations move from cart to order without double counting.
  - Idempotency keys are scoped per user and locked against concurrent duplicates.
- **Tested against the real thing.** 100 test functions (Postgres and Redis via testcontainers, about 15 s) plus an HTTP end-to-end script that runs the whole money flow against a live Blnk ledger and checks balances.

## How money moves

```
Buyer checkout (verified phone + delivery address)
  total = goods + merchant's flat delivery fee (free above the shop's threshold)
  └─► order pending_payment ─► Chapa hosted checkout
Chapa webhook / callback / buyer refresh / expiry job
  └─► verify with Chapa (status, amount, mode, charge)
  └─► Blnk  @World ─inflight─► Escrow                         order paid, stock committed
Merchant: processing ─► shipped ─► auto-confirm scheduled (7 days)
Buyer confirms delivery (or auto-confirm) ─► settle_order job
  └─► Blnk  commit escrow
            Escrow  ─► Merchant          goods − 5% fee + delivery
            Escrow  ─► Revenue           5% platform fee
            Revenue ─► Processing fees   Chapa's charge (absorbed by the platform)
Buyer disputes ─► escrow frozen ─► admin: release (settle) or refund
Payout request (verified shop + KYC, limits in Addis Ababa time)
  └─► Blnk  Merchant ─inflight─► @World   (held; cannot be double-spent)
Admin approves ─► Chapa transfer ─► verify ─► commit (void on failure)
```

The full design is in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), including the ledger reference table, order state machine and job catalogue.

## Tech stack

| Concern | Choice |
|---|---|
| Language / HTTP | Go 1.26, chi v5 |
| Data | PostgreSQL 16 (pgx v5, sqlc, goose), Redis 7 |
| Ledger | Blnk 0.15 (self-hosted, double-entry, inflight holds) |
| Payments | Chapa: checkout, verify, refunds, transfers |
| Background jobs | River (Postgres-backed, transactional enqueue, periodic jobs) |
| SMS | AfroMessage |
| Auth | RS256 JWT + rotating refresh tokens, Argon2id passwords, SMS codes |
| Tooling | testcontainers-go, gomock, golangci-lint v2, swag, Docker Compose, GitHub Actions |

## Quick start

Requirements: Go 1.26+, Docker, and `jq`, `curl` and `openssl` for the end-to-end script.

```bash
cp .env.example .env          # development defaults: Chapa simulated in memory, SMS printed to the log
make generate-keys            # RSA keys for JWT
make local                    # Postgres, Redis, Blnk server + worker
make seed-blnk                # create the ETB ledger and platform balances; writes their IDs into .env
make run                      # API on :8080 (runs migrations on start)
./scripts/e2e-money-flow.sh   # sign-up → SMS verify → shop → checkout → settlement → dispute → payout
```

- Swagger UI: http://localhost:8080/swagger/index.html
- Health: http://localhost:8080/health (Postgres, Redis and ledger status)

In development, `CHAPA_MODE=mock`, `SMS_PROVIDER=log` and `OTP_DEV_CODE=123456` make the flow scriptable; config refuses all three outside development. To use the Chapa sandbox, set `CHAPA_MODE=live`, a `CHASECK_TEST-…` key, the webhook secret hash from the Chapa dashboard, and a public `CHAPA_CALLBACK_URL`. Every setting is documented in [.env.example](.env.example) and [.env.production.example](.env.production.example).

## API overview

All routes are under `/api/v1`; Swagger has the full contract. Endpoints that move money require an `Idempotency-Key` header.

| Area | Routes |
|---|---|
| Auth | `POST /register`, `/login`, `/refresh`, `/logout`, `/logout-all`, `/auth/otp/send`, `/auth/otp/verify`, `/auth/password/reset/{request,confirm}` |
| Catalog (public) | `GET /products?q=&category_id=&merchant_id=`, `/products/{id or slug}`, `/categories`, `/shops/{slug}` |
| Cart (guest or signed in) | `GET/DELETE /cart`, `POST /cart/items`, `PUT/DELETE /cart/items/{variantId}`, `POST /cart/merge` |
| Buyer | `/addresses…`, `POST /orders/checkout`, `GET /orders[/{id}]`, `POST /orders/{id}/{cancel,confirm-delivery,dispute,payment/refresh}` |
| Merchant | `POST /merchant/onboard`, `GET /merchant/dashboard`, `/merchant/products…` (any status, variants, stock, images), `/merchant/orders…` (with the delivery address; `status`, `ship`, `cancel`) |
| Wallet | `GET /merchant/wallet[/transactions,/banks,/payouts]`, `POST /merchant/wallet/payout` |
| Admin | users and KYC, merchant verification and wallets, `POST /admin/orders/{id}/resolve`, payouts `process`/`reject`, `GET /admin/reconciliation` |
| Payments | `POST /webhooks/chapa` (signature verified), `GET /payments/chapa/callback` |

## Development

```bash
go test -short ./...   # unit tests, no Docker
make test              # all tests with -race (integration tests use testcontainers)
make lint              # golangci-lint v2
make sqlc-generate     # after editing sqlc/queries or migrations
make mocks             # after changing repository/service interfaces
make swagger           # after changing handler annotations
```

```
cmd/api            entrypoint and dependency wiring
internal/service   business rules: checkout, payment, order, wallet, cart, OTP, reconciliation
internal/jobs      River job types, workers, transactional enqueuer
internal/http      router, handlers, middleware (auth, idempotency, rate limit, client IP)
internal/blnk      ledger client, idempotent by reference
internal/chapa     Chapa client + in-memory fake
internal/repository, internal/db   data access (sqlc)
pkg/phone, pkg/sms Ethiopian phone normalization, AfroMessage sender
migrations         goose migrations (also the sqlc schema)
scripts            Blnk setup, seed data, end-to-end money-flow test
```

## Documentation

| Document | What's in it |
|---|---|
| [Architecture](docs/ARCHITECTURE.md) | Components, ledger model, order states, payment sequence, jobs, security |
| [Design decisions](docs/DECISIONS.md) | Why River and not RabbitMQ, escrow timing, idempotency, fees, disputes, … |
| [Engineering log](docs/ENGINEERING_LOG.md) | What the audit found, how it was fixed, bugs only the real ledger caught |
| [Business analysis](docs/BUSINESS_ANALYSIS.md) | Unit economics, risks, a sharper wedge, pilot kill criteria |
| [Roadmap](docs/ROADMAP.md) | Status, launch checklist, next features |
| [Original spec](docs/original-spec.md) | The March 2026 design the project started from |

## Status

The Phase 1 MVP is complete and passes its end-to-end test against Blnk with the Chapa mock. It has not yet been run against the live Chapa sandbox or AfroMessage, and a production launch needs an NBE-compliant escrow arrangement. See the [roadmap](docs/ROADMAP.md).

## License

[MIT](LICENSE) © 2026 Natnael Alemayehu
