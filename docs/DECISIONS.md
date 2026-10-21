# Design decisions

A running record of decisions that shaped the system, newest first. Each entry gives the choice, the alternatives considered, and why.

## D-014 · Keep River instead of RabbitMQ (Oct 2026)

**Choice:** River, a Postgres-backed job queue, for all background work.
**Alternatives:** RabbitMQ with a transactional outbox and relay; or both.
**Why:** The consistency model depends on inserting a job in the same transaction as the change that needs it, such as "order delivered" and "settle order". RabbitMQ can't take part in a Postgres transaction, and it can't reliably schedule work hours or days ahead, which payment expiry (1 h), auto-confirm (7 days) and cart expiry (7 days) all need. Switching would mean an outbox table, a relay, and Postgres-side scheduling anyway, plus one more service to operate. Revisit when the system splits into services or needs event fan-out to other consumers.

## D-013 · The platform absorbs Chapa's processing charge (Oct 2026)

**Choice:** Buyers pay the listed price, and merchants receive subtotal − platform fee. Chapa's charge is booked from Revenue to a Processing-fees balance (`chapa-fee-<order>`).
**Why:** Simple, predictable prices for buyers and merchants, with margins kept visible in the ledger. Chapa doesn't return its charge on refunds, so refunds and reversals also book it, idempotently per order. This leaves the platform with about 1–1.5% of GMV net; see [BUSINESS_ANALYSIS.md](BUSINESS_ANALYSIS.md).

## D-012 · Escrow only, no cash on delivery, in the MVP (Oct 2026)

**Why:** Cash on delivery breaks the escrow promise and the ledger's completeness, and needs delivery partners who collect cash. Escrow is the product's differentiator; cash on delivery can come later.

## D-011 · Merchant flat delivery fee (Oct 2026)

**Choice:** Each merchant sets a flat fee and an optional free-delivery threshold. The fee is added to the order total, held in escrow, and passed to the merchant in full, with no platform fee on it.
**Alternatives:** Per-subcity pricing; no delivery fee.
**Why:** Small merchants deliver themselves or through a known courier at a fixed price. Per-zone pricing can come once demand shows it's needed.

## D-010 · Phone-first identity with SMS verification (Oct 2026)

**Choice:** Normalize every Ethiopian mobile format to `+2519…`/`+2517…`, verify by SMS code through AfroMessage, and reset passwords by SMS. A verified phone is required to check out and to open a shop.
**Why:** Many users have no email address, and `0911…` and `+251911…` must not become two accounts. Chapa also requires the local `09…` form. Verified phones cut fraud and give delivery a working contact.

## D-009 · Disputes stop at delivery confirmation; auto-confirm after N days (Oct 2026)

**Choice:** Buyers can dispute while an order is `processing` or `shipped`. Confirming delivery, or the auto-confirm after 7 days, releases escrow. Admins resolve disputes with `release` or `refund`.
**Why:** Without auto-confirm, a buyer who never confirms freezes the merchant's money forever. Allowing disputes after confirmation would race the settlement job.

## D-008 · Payout gating (Oct 2026)

**Choice:** Payouts require a verified shop and a KYC-verified owner. Mobile-money destinations must be Ethiopian numbers, bank codes must come from Chapa's list, and there's a minimum plus a daily cap per Addis Ababa calendar day.
**Why:** Payouts are the step where fraud turns into real loss, and NBE expects KYC.

## D-007 · Payment reversals require the payload HMAC (Oct 2026)

**Choice:** `charge.reversed` and `charge.refunded` webhooks are acted on only when `x-chapa-signature`, the HMAC of the exact payload, verifies. Before settlement they cancel the order; after settlement they debit the merchant, with overdraft allowed.
**Why:** Normal payment events are always re-verified with Chapa's API. Reversals take money back from merchants, so they need the stronger signature; Chapa's other header is a static value.

## D-006 · Daily ledger reconciliation (Oct 2026)

**Choice:** A periodic job and an admin endpoint compare every merchant wallet in Blnk with that merchant's `wallet_transactions`, and the escrow balance with paid orders not yet settled.
**Why:** With money in two systems, drift has to be caught the same day, not discovered by a merchant.

## D-005 · Commit and void inflight with `skip_queue` (Oct 2026)

**Why:** A review of Blnk's source showed that 0.15 queues inflight commits by default. Settlement would then try to credit the merchant from an escrow balance whose commit hadn't applied yet. Skipping the queue makes the order deterministic. "Already queued" (`GEN_CONFLICT`) snoozes the job.

## D-004 · Escrow is created when payment is verified, not at checkout (Oct 2026)

**Alternatives:** The original spec put the escrow inflight in checkout.
**Why:** The ledger should only show money that actually arrived. It also removes a failure mode from checkout, and expired checkouts leave nothing in the ledger to clean up.

## D-003 · Idempotency by deterministic Blnk references (Oct 2026)

**Why:** Jobs retry. Deriving every reference from the order or payout ID (`sale-credit-<order>`) means a retry after a lost response returns the existing transaction instead of paying twice. The database side uses a unique index on `wallet_transactions.blnk_transaction_id`.

## D-002 · Transactional job enqueue through an `Enqueuer` interface (Oct 2026)

**Why:** It replaced a global River client. Services depend on `jobs.Enqueuer`, which the tests implement with `NoopEnqueuer`, and the River client is bound after the workers are constructed. This breaks the services → workers → services construction cycle.

## D-001 · Cart reservations transfer to the order (Oct 2026)

**Why:** The original code reserved stock again at checkout and leaked the cart's reservation. Now the cart owns the reservation until Chapa accepts the payment. Then the cart lines are cleared without releasing, so the order owns it. If Chapa fails, the order is cancelled and the cart is untouched.
