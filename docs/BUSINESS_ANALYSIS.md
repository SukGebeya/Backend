# Business analysis

An assessment of whether Gebeya is worth pursuing, written in October 2026 once the MVP worked end to end. Figures are rough estimates for illustration, not market research.

**Verdict: 5/10 as scoped. About 6–7/10 if narrowed to the wedge below.** It's worth a cheap, time-boxed pilot; it isn't worth building out as a general marketplace.

## Strengths

- **Real pain.** Online selling in Ethiopia happens largely in Telegram channels, TikTok and Facebook, and scams are common in both directions. Escrow, where the buyer's money is held until delivery, addresses that directly.
- **The rails exist now.** Telebirr reaches tens of millions of users and Chapa provides licensed payment collection and transfers.
- **The long-term value is data.** Every merchant builds a verified sales history in the ledger, which is exactly what a working-capital product (phase 2) needs. Lending earns far more than marketplace fees.
- **The hard part is built.** Escrow, reconciliation, payouts and disputes are already implemented and tested.

## Weaknesses

### 1. Thin margin per order

The end-to-end run kept about 70 ETB of platform revenue on a 5,150 ETB order. A typical order:

| | ETB |
|---|---|
| Goods 2,500 + delivery 150 | 2,650 paid |
| Platform fee (5% of goods) | +125 |
| Chapa charge (~3% of total) | −80 |
| SMS, refunds, disputes, support | −10 to −20 |
| **Net** | **≈ 25–35 (about 1–1.5% of goods)** |

A small team in Addis might cost around 1–1.5M ETB a month, which means roughly 35–50k orders a month just to break even.

### 2. Building buyers and sellers at once

A destination marketplace has to attract both sides, against free channels people already shop in, and against Zemen Gebeya, which has Ethio Telecom and Telebirr distribution.

### 3. Cash on delivery competes with escrow

In Addis, buyers already handle trust by paying on delivery. Escrow wins where cash on delivery can't work: between cities, for high-value items, and between strangers.

### 4. Regulation

Holding buyer funds and merchant balances may count as stored value under NBE directives, and lending needs a licence or a partner bank or microfinance institution. This is the largest risk and must be settled before launch.

### 5. The economy

Inflation and the weakened birr since the 2024 float squeeze spending, and delivery outside Addis is unreliable.

## The sharper wedge

1. **Escrow payment links for existing social sellers.** A seller pastes a Gebeya link into a Telegram or TikTok post; the buyer pays into escrow; the seller is paid on delivery. This avoids building a buyer audience, and the current backend covers most of it. The missing pieces are a product link that skips the cart and a Telegram bot.
2. **One vertical, one corridor.** For example, phones and electronics between Addis and regional cities, where order values and scam risk are high and cash on delivery is impossible.
3. **Charge both sides.** A visible 2–3% buyer protection fee plus a 3–5% merchant fee roughly doubles the per-order margin.
4. **Payments as the funnel, lending as the business.** Talk to a microfinance institution or bank partner early; six months of verified sales history is what makes merchant advances underwritable.

## Pilot and kill criteria

Run a 90-day pilot with 30–50 sellers. Pursue it if it reaches:
- about 1,000 escrow orders a month
- over 30% of buyers buying again
- disputes under 3%
- sellers who keep using it once they're charged

Stop or pivot to a payments-only tool if sellers won't pay roughly 5% or more, or buyers keep choosing cash on delivery.
