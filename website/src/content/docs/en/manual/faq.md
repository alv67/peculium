---
title: FAQ
order: 9
description: Short answers to the questions that come up most, each pointing to the chapter that goes deeper.
---

Short answers first, links for the full story. If your question is not here,
the sidebar lists the rest of the manual.

## I can't log in — my account is "pending"

Registration may require approval: if the server has auto-approve off, you
get "Registered! An administrator must approve your account" and sign-in
stays refused until it happens. An administrator approves you under
**Admin → Users** (the row shows the Pending badge) — see
[Administration](/manual/administration). A disabled account needs the same
page: an admin **re-enables** it.

## Prices aren't updating — I see a rate-limit warning

Yahoo Finance throttles heavy request traffic, and Peculium shows it honestly
in the strip under the header ("Some prices not updated (Yahoo rate limit)").
There is nothing to fix on your side: the limit passes, and you can
pull fresh quotes on demand with **Refresh prices**. The
[Data & sync](/manual/data-sync) chapter — and the admin **Data & Sync**
health page — show what happened and when.

## An asset shows "no price" and its value is carried at cost

That asset's price source is **No price** (or **Manual price** with no entry
yet). Carried at cost means the position is valued at what you paid, so its
P/L reads zero instead of guessing. Set the source to **Yahoo Finance** for
anything Yahoo quotes, or record dated prices by hand under Manual — both
live on the asset's **Data** tab; see [Data & sync](/manual/data-sync) and
the asset model in [Concepts](/manual/concepts).

## A position is excluded with "missing FX rate"

Your consolidated dashboard needs a conversion rate from the holding's
currency to your **base currency**; when no rate is available, the amount is
flagged and excluded rather than silently mis-converted. Fix it by managing
that currency (Settings → **Currencies** whitelist) — the mechanics are in
[Currencies and the base currency](/manual/concepts#currencies-and-the-base-currency).

## I forgot my password

Self-hosted instances don't email reset links. If you are signed out, an
administrator can **reset your password** from **Admin → Users**; you should
change it right after signing in. While you can still log in, set a new one
yourself under Settings → **Password** ([Settings](/manual/settings),
[Administration](/manual/administration)).

## Add or replace when restoring a backup?

**Add to current data** is non-destructive: the backup's portfolios are
imported as new ones and nothing you have is touched. **Replace current
data** deletes your existing portfolios and transactions first (assets are
never deleted — the catalogue is shared). Read
[Backup & restore](/manual/backup) before using replace.

## Where is my data stored — and how do I move servers?

On your own machine, in the database inside Peculium's stack: it never
leaves your network except as market data fetched from Yahoo. To move to a
new server, bring the stack up with the [install guide](/install), then
either restore each user's JSON **backup** on the new instance or — for a
complete migration including users — move the admin
**Server backup** dump. Details in [Backup & restore](/manual/backup).

## Can more than one person use Peculium?

Yes: it is multi-user by design, with real accounts and per-user
workspaces — each person keeps their own portfolios. Roles
(**Admin** / **Editor** / **Viewer**) decide who can manage the instance,
and registrations are approved (or auto-approved) by an administrator. The
whole picture is in [Administration](/manual/administration).
