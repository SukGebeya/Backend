# Engineering log

A chronological account of how the backend got from a feature-complete-looking prototype to a working MVP: what was found, why it mattered, and how it was fixed. It's written as source material for a write-up, so it records findings and numbers, not just changes.

## Starting point (March 2026)

The project began from a generic Go REST boilerplate (chi, pgx, sqlc, goose, JWT, Redis). It was built out phase by phase against [original-spec.md](original-spec.md): auth, merchant onboarding with a Blnk wallet, catalog, carts, Chapa checkout, River jobs, orders and payouts. It was about 20k lines and compiled, and every phase checkbox was ticked.

## October 2026, session 1: recovery and audit

### The workspace was empty

The local checkout had been replaced by 818 zero-byte files, including `.git` internals, from a copy that went wrong on 2026-09-04. GitHub `main` still had the last pushed state, so the repository was re-cloned. About 15 files that had only ever existed locally were lost (a payment service, webhook-event storage, an audit-log package, and two docs) and were rebuilt later.

**Lesson:** push early, even to a WIP branch.

### What the audit found

The code built, but reading it end to end against how money actually has to move turned up bugs that would have lost money or stock in production:

| Area | Finding | Impact |
|---|---|---|
| Ledger | `GetBalance` multiplied Blnk's balance by 100, but Blnk already returns minor units | Wallets showed 100× the real balance; payouts were checked against that figure |
| Payments | The payment job marked orders paid from the webhook body without calling Chapa's verify API | A forged or replayed webhook could mark an order paid |
| Payments | The webhook signature was checked with the API secret instead of the webhook secret, and payloads were parsed as nested `data.*` when Chapa sends flat fields | Real webhooks would have been rejected or misread |
| Payouts | The payout job debited Blnk and marked the payout "completed" without ever calling Chapa's transfer API | Money never left; merchants were told it had |
| Payouts | No hold on request | Several requests could exceed the balance |
| Settlement | Three Blnk calls in a row on delivery confirmation; a retry failed at "already committed" | A failure partway through could not be recovered |
| Stock | Reserved at add-to-cart and again at checkout; the cart's reservation was never released | Phantom out-of-stock; stock leaked permanently |
| Stock | Reserve/commit queries were `:exec`, so insufficient stock affected 0 rows and returned success | Overselling |
| Stock | Expired carts and unpaid orders never released stock | Stock leaked permanently |
| Cart | Logged-in users got guest session carts | Checkout returned "cart not found" for every signed-in user |
| Cart | Merging a guest cart reused one ID for every inserted row | Merging two or more new items failed |
| Security | The idempotency cache wasn't scoped per user, and its TTL was passed in nanoseconds but read as seconds | One user could receive another user's checkout response; entries lived for millennia |
| Security | `chi.RealIP` trusted any `X-Forwarded-For`, and the rate limiter keyed on IP:port | Rate limiting could be bypassed with a spoofed header or a new connection |
| Infra | `docker-compose` put Blnk on another network and port, with no Blnk worker, and passed `redis://` URLs as host:port | The stack couldn't start |
| Infra | `.golangci.yml` used v1 keys under `version: 2` | Every lint setting and exclusion was silently ignored (958 findings) |
| Tests | Each test started its own Postgres container | 8+ minutes; failed without local key files |

### How it was fixed

These are the design changes; [DECISIONS.md](DECISIONS.md) records why each was chosen.

- **Payments are re-verified.** Every payment path goes through `PaymentService.ProcessPayment`, which calls Chapa verify and checks the amount. Escrow is created only after that verification succeeds.
- **Idempotent ledger.** Every Blnk transaction has a reference derived from the order or payout, and a duplicate-reference error returns the existing transaction.
- **Settlement and refunds became retryable River jobs.** Each step is idempotent, so a retry continues where it stopped.
- **Payouts hold funds first.** The request places an inflight hold in Blnk, approval claims the payout atomically, and the job sends the Chapa transfer and then commits or voids the hold.
- **Transactional jobs.** Jobs are enqueued in the same database transaction as the change that needs them.
- **Race-safe status changes.** Order status uses compare-and-set transitions.
- **Tests got fast.** One container per package with a cloned template database per test: 8 minutes became about 15 seconds.

### Bugs only the real ledger caught

The service tests mock Blnk. A shell script (`scripts/e2e-money-flow.sh`) that drives the full flow over HTTP against a real Blnk 0.15.6 found three more problems:

1. Blnk rejects transactions that send both `amount` and `precise_amount`. Fixed by sending only `precise_amount`.
2. Blnk 0.15 queues inflight commits by default, so settlement could try to credit the merchant before the escrow commit landed. Fixed with `skip_queue: true`; this was found by reading Blnk's source after the mock-based tests passed.
3. Blnk ignores its API key unless `BLNK_SERVER_SECURE=true` is set. With only the key configured, the ledger was open to anyone on the network.

**Lesson:** mocks verify your assumptions about a dependency, not the dependency. Keep at least one test against the real thing.

## October 2026, session 2: making it work for Ethiopia

Business decisions taken in this session: keep River rather than RabbitMQ, no cash on delivery, the platform absorbs Chapa's fee, and merchants set a flat delivery fee.

| Problem | Change |
|---|---|
| Chapa requires `phone_number` as `09xxxxxxxx`/`07xxxxxxxx`, but users were stored as `+251…`, so live checkout would fail for any buyer with a phone | `pkg/phone` normalizes every format; Chapa gets the local format |
| `0911…` and `+251911…` were different accounts | Normalized at register, login, addresses, onboarding and payouts |
| No way to verify phones or reset a password without email | SMS codes through AfroMessage (Amharic and English), password reset by SMS |
| Chapa's charge is non-refundable and wasn't in the ledger | Booked once per order to a processing-fees balance, including on refunds |
| No delivery fee | Merchant flat fee with an optional free-delivery threshold, held in escrow and paid to the merchant |
| Buyers who never confirm freeze escrow forever | Auto-confirm after 7 days; disputes freeze it until an admin resolves them |
| Chargebacks ignored | Reversal webhooks (payload HMAC required) cancel the order or claw back from the merchant |
| Anyone could withdraw | Payouts need a verified shop and KYC, a valid destination for the method, a minimum and a daily cap |
| No ledger cross-check | Daily reconciliation at 03:00 Addis Ababa time, plus an admin endpoint |
| Product lists had no price, stock or images; `total` was the page size; slugs were ambiguous across shops | One aggregate browse query with Amharic-safe search; global slugs; image URLs |
| Deleting a product that had sold violated order foreign keys | Delete archives the product and deactivates its variants |
| Rate limiter returned 500 when Redis was down | Fails open with a log line; auth endpoints get a stricter limit |
| Deactivated users could still log in | Blocked at login and refresh |

**Result:** the e2e script covers phone sign-up, SMS verification, an Amharic shop and product, checkout with delivery, settlement (merchant receives 4,900 ETB; revenue keeps the 250 ETB fee minus Chapa's 180.25 ETB), a dispute leading to a refund, payout gating and completion, and reconciliation. All 9 merchant wallets reconciled.

## October 2026, session 3: a frontend

A SvelteKit 3 / Svelte 5 web app now lives in a sibling repo, [SukGebeya/Frontend](https://github.com/SukGebeya/Frontend) (decision D-015). Its Playwright test drives a seller and a buyer through the same flow as `scripts/e2e-money-flow.sh`, in the browser: sign-up with SMS code, onboarding, listing a product, paying through the shared link, shipping, confirming delivery, settlement into the wallet, and a payout request.

Building real screens against the API surfaced gaps the curl-based script never hit:

| Found | Fix |
|---|---|
| The seller's order had only `shipping_address_id`, and addresses are readable only by their owner, so a seller could not see where to deliver | Order details include `shipping_address` (`OrderService.WithAddresses`) |
| The public product page had no shop, so buyers could not see the seller, the verified badge or the delivery fee before paying | `GET /products/{slug}` includes `shop`; `ShopResponse` gains `merchant_id` so a shop page can list its products |
| Sellers had no way to read a draft product or a hidden option | `GET /merchant/products/{id}` returns any status and all options |
| `GET /merchant/products` serialized a Go struct without JSON tags (`Items`, `TotalPages`), unlike every other list | Mapped to the standard paginated response |
| Behind the frontend, every user shared one rate-limit bucket | `TRUSTED_PROXY_COUNT=1` with the frontend forwarding the client address (D-016) |

Frontend findings worth recording:
- swag marks every response field optional; the frontend's codegen reads the Go structs and marks fields required unless they are `omitempty`.
- Svelte's `value={…}` overwrote what people typed before the page hydrated; inputs use `defaultValue` instead, which matters on slow mobile connections.
- `pattern="[0-9]{6}"` in Svelte markup becomes `[0-9]6` (the braces are an expression), which silently blocked the SMS code form.

## October 2026, session 4: every endpoint end to end, with a mock Chapa

The goal: every endpoint working and every important piece of data flowing end to end, with no Chapa account. Chapa, including Telebirr, is now an HTTP mock served by the API (D-017), and the real client talks to it. `scripts/e2e-endpoints.sh` calls all 83 routes, checks the data each returns, and fails unless every route returned a 2xx at least once (it compares its calls with `go run ./cmd/routes`).

Its payment scenarios:
- A Telebirr payment that is shipped and delivered.
- A card payment the buyer cancels and a CBE Birr payment the seller cancels, both refunded at the mock.
- A dispute released to the seller.
- A declined payment, and an abandoned one the buyer then cancels. Stock ends where it started.
- Three payouts: one completed, one rejected by the admin, and one that fails at the provider and returns to the wallet.
- A chargeback after settlement, clawed back from the merchant's wallet.

`scripts/e2e-money-flow.sh` now pays through the hosted checkout too, instead of signing its own webhook.

| Found | Fix |
|---|---|
| `PUT /users/me` with only `full_name` was rejected ("at least one of email or phone is required") | Any one of email, phone or full_name is enough |
| Updating an option without a title (the frontend's Hide/Show button) blanked the title | `title` is optional on update |
| Deleting a product or option only archived or hid it, even if it had never sold | Deleted for real; archived or hidden only when order items reference it (FK violation → `repository.ErrStillReferenced`) |
| Add-to-cart and quantity-change responses had an empty `product_name` | `CartService.DescribeItem` fills it the same way `GET /cart` does |
| Payouts were re-checked only after a fixed two minutes | `PAYOUT_RECHECK_INTERVAL` (2m by default, 10s in `.env.example`) |
| The auth rate limit (10 a minute) blocked any scripted run longer than a few sign-ins | `AUTH_RATE_LIMIT_REQUESTS` (10 by default, 60 in `.env.example`) |

## Numbers

| | Before | After |
|---|---|---|
| Test suite runtime | 8+ min, failing | ~15 s, passing with `-race` |
| Test functions | ~30 | 113 |
| golangci-lint findings | 958 (config ignored) | 0 |
| Migrations | 16 | 22 |
| HTTP routes | ~60 | 83 |

## Open questions to write about

- Whether a ~1–1.5% net take rate can carry a marketplace; see [BUSINESS_ANALYSIS.md](BUSINESS_ANALYSIS.md).
- Escrow and NBE licensing: who may legally hold buyer funds.
- Payment links for social sellers versus a destination marketplace.
