---
title: Data & sync
order: 5
description: Where prices come from, what the warnings mean, and how to tell a healthy sync.
---

Every chart is only as good as the prices behind it. This chapter covers
where they come from, what happens when they don't, and how to check the
sync. The currency side of the story is in [Concepts](/manual/concepts); the
asset page itself is covered in [Daily use](/manual/daily-use).

## Where prices come from

Quotes come from **Yahoo Finance**. Peculium keeps a local price history,
refreshes the latest quotes in the background, and tells you how fresh the
numbers are: the header carries a **"Prices as of {time}"** stamp, and the
consolidated values are computed with exactly those prices. Need them right
now, not later? Use **Refresh prices** to pull fresh quotes on demand.

![The app header with the freshness stamp](/screenshots/en/manual/data-header.png)

*The app header — the "Prices as of" stamp says which prices your numbers use.*

## Data quality

When a fetch doesn't succeed, a strip of chips appears under the header —
each one names the problem and links to where you can fix it: some prices
were not updated (Yahoo **rate limit**), a number of price updates failed, or
an amount is **excluded — missing FX** for a handful of holdings. Excluded
beats invented: a holding without a usable rate is left out and flagged,
never silently mis-converted. An asset with no price at all shows as
**no price**: its value is **carried at cost** — what you paid — so its
P/L reads zero instead of guessing.

## Price history & backfill

Each asset page charts its stored **price history**. When history is thin or
missing — a freshly registered asset, a gap in the records — run **Backfill
full history** from the asset's menu (or its **Data** tab): Peculium pulls
the complete daily history from Yahoo and the chart fills in. The same menus
carry **Update from Yahoo**, which refreshes the asset's metadata rather than
its prices.

![An asset's price-history chart](/screenshots/en/manual/data-price-history.png)

*Price history — backfill pulls the whole series in one go.*

## Price source per asset

The asset's **Data** tab decides where its prices come from:

- **Yahoo Finance** — the default: automatic quotes and backfill. Use it for
  anything Yahoo lists by ticker.
- **Manual price** — record dated prices by hand, for assets without an
  automatic feed (an unlisted fund, a relative's fixed-rate bond).
- **No price** — opt the asset out: it stays carried at cost, valued at what
  you invested.

## The Data & Sync page (admins)

Administrators get the sync's own health report under **Data & Sync**: the
**Success Rate** with totals of **successes**, **failures** and **rate
limited** calls, and the **Recent Events** table — timestamp, type, status,
code, message, duration. Choose the window with the period selector (**Today
/ Last 24h / Last 100**) and press **Refresh Now** to re-poll the report.
When a quality chip keeps coming back, this page tells you whether Yahoo is
grumbling generally or one specific asset is failing.

![The Data & Sync health page](/screenshots/en/manual/data-sync-health.png)

*Data & Sync — success rate, totals and recent events for the price sync.*

A quiet sync means honest charts: check [Dashboard &
analytics](/manual/dashboard) for what the numbers then tell you about your
portfolios.
