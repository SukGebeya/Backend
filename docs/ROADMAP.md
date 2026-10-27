# Roadmap and status

Last updated: 2026-10-27.

## Status

The Phase 1 MVP is implemented. `go test -race ./...` passes, golangci-lint reports 0 issues, and `scripts/e2e-money-flow.sh` passes against Blnk 0.15.6 (API auth on). `scripts/e2e-endpoints.sh` calls all 83 routes and checks their data. Both pay through the mock Chapa (D-017), whose hosted checkout offers Telebirr, CBE Birr and a test card, and which also handles refunds, payouts and chargebacks. The Chapa sandbox and AfroMessage have **not** been tested live.

A SvelteKit web app ([SukGebeya/Frontend](https://github.com/SukGebeya/Frontend)) covers the buyer and seller flows in Amharic and English; its Playwright test runs the same money flow through the browser. Admin work is still done in Swagger.

## Before launch

- [ ] **Regulation:** settle the escrow arrangement with Chapa and NBE. Buyer funds held until delivery must sit with a licensed partner.
- [ ] **Tax:** VAT registration for platform fees; decide how merchant TINs are used.
- [ ] **Chapa sandbox** (`CHASECK_TEST-…`):
  - webhook headers and the `charge.reversed` payload
  - `charge` and `mode` in the verify response
  - the Telebirr and CBE Birr entries returned by `/banks`
  - refund timing
- [ ] **AfroMessage:** sender name approval; one live OTP send.
- [ ] **Production secrets:** `OTP_SECRET`, `BLNK_SECRET_KEY`, `CHAPA_WEBHOOK_SECRET`, JWT keys, Postgres and Redis passwords.
- [ ] **Hosting:** TLS, backups for both Postgres databases (app and Blnk), log shipping, and an alert on `RECONCILIATION MISMATCH` log lines.
- [ ] **Frontend hosting:** run it behind the same reverse proxy as the API, set the API's `TRUSTED_PROXY_COUNT=1` (D-016), and point `CHAPA_RETURN_URL` at the frontend's `/payment/complete`.

## Next, if the pilot goes ahead

See [BUSINESS_ANALYSIS.md](BUSINESS_ANALYSIS.md) for why these are the next steps.

1. **Payment links:** the frontend's product link plus "Buy now" covers the basic case by clearing the cart first. A checkout endpoint that takes a variant directly would remove that workaround.
2. **Telegram bot:** sellers create links and see orders; buyers get updates.
3. **Buyer protection fee:** a configurable buyer-side fee shown at checkout and booked to revenue.
4. **Image storage:** pre-signed uploads to object storage (today images are URLs).
5. **Merchant advances (phase 2):** underwriting from ledger history, with a lending partner.

Smaller follow-ups found while building the frontend:
- **Snapshot the delivery address on the order.** Orders reference `addresses` with `ON DELETE SET NULL`, so a buyer who deletes an address leaves the seller without one.
- **Per-order Chapa `return_url`** (`…/payment/complete?order=<id>`). Today the frontend remembers the order in a cookie, which fails if the buyer finishes paying on another device.
- **Admin pages** for KYC, shop verification, disputes and payout approval.
- **A status filter** on `GET /merchant/orders`.
- **Clearing a product description.** `PUT /merchant/products/{id}` ignores empty fields, so a seller can replace a description but not remove it.

## Not planned for the MVP

Cash on delivery, multi-merchant split checkout, Fayda national ID verification, VAT invoices, Amharic API error messages (the frontend translates its own text, but API messages are English), RabbitMQ.
