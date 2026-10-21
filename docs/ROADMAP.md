# Roadmap and status

Last updated: 2026-10-21.

## Status

The Phase 1 MVP is implemented. `go test -race ./...` passes, golangci-lint reports 0 issues, and `scripts/e2e-money-flow.sh` passes against Blnk 0.15.6 (API auth on) with the Chapa mock. The Chapa sandbox and AfroMessage have **not** been tested live.

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

## Next, if the pilot goes ahead

See [BUSINESS_ANALYSIS.md](BUSINESS_ANALYSIS.md) for why these are the next steps.

1. **Payment links:** a shareable product or checkout link that skips the cart, for Telegram and TikTok sellers.
2. **Telegram bot:** sellers create links and see orders; buyers get updates.
3. **Buyer protection fee:** a configurable buyer-side fee shown at checkout and booked to revenue.
4. **Image storage:** pre-signed uploads to object storage (today images are URLs).
5. **Merchant advances (phase 2):** underwriting from ledger history, with a lending partner.

## Not planned for the MVP

Cash on delivery, multi-merchant split checkout, Fayda national ID verification, VAT invoices, Amharic API error messages, RabbitMQ.
