# Release Notes

## Unreleased

### Features
- The "Data & Sync" page now shows the sync jobs and their outcome — queued, running, completed, partially failed or failed — with progress, the real duration (no longer always zero) and the error when there is one, refreshed automatically while something is running; a long download shows its own start and finish with the result even if it runs out of time
- Every history download and split check now appears in "Data & Sync", one entry per asset, whether it succeeded or failed, so you can see exactly which assets were updated and when
- In "Data & Sync" each job now names its target — the asset ticker or the portfolio name — instead of an opaque code
- The "Data & Sync" page lets an admin clear all sync activity (jobs and their log entries) with one confirmed action, and activity older than 90 days is removed automatically, so the page never grows without bound
- The app now shows which build it is running: the version and build time sit discreetly at the bottom of the sidebar (and on the login page), and a new "About this app" card in Settings → Preferences lists the version, the commit (linked on GitHub) and the build time, with a button to copy it when reporting an issue
- Market data now downloads in the background instead of making you wait: updating prices, re-downloading an asset's full history or metadata, and simply opening the app or a portfolio all return immediately, and the pages update themselves as soon as the data is ready — so a slow connection or a large collection no longer makes an operation hang or fail silently
- The asset page now shows the state of its price history: while it is still downloading you see a loading state instead of an empty chart (updated automatically), when there is no history yet you get a button to download it, and when the history is only partially downloaded a note tells you so, with the same button to complete it
- The assets list now shows each asset's last close with its date, and the asset page labels that date as "data as of": since markets close at different times, each asset's latest value can be dated differently, and that is normal
- The price-update warning now distinguishes a full success from a partly failed update, and the full-history download tells you when the price history was updated but the stock splits could not be retrieved

### Fixes
- The latest price and the per-period returns are now the same on every installation that refreshes at the same time, instead of sometimes differing
- Restoring a backup now downloads the market data for the restored assets right away, instead of leaving their charts empty until the next time the app is opened

## v1.0.0 — 30 Sep 2026 (first release under the Peculium name)

### Features
- Admins can now back up and restore the whole server database from the new **Admin → Server backup** page: download a complete database dump, and restore the instance from one — the destructive replace asks you to type an explicit confirmation, and the app reminds you that every account is replaced, so you may need to log in again
- New Settings → Backup & restore page: download all of your data — portfolios, transactions, assets with their exposure and manual prices — as a single JSON file, and restore it choosing whether to **add** it to your current data or **replace** it (replace asks for an explicit confirmation first, and your assets are never deleted)
- New **Admin area** for admin accounts: a **Users** page to review every account with its role and status, approve pending registrations, disable and re-enable accounts, change roles, and set a new password for a user who lost theirs; plus a **Server settings** page. On a brand-new server the first registered account becomes the administrator
- New server setting **Auto-approve new registrations** (on by default): when it is off, new sign-ups stay pending until an administrator approves them
- The login page now explains exactly what is going on when an account is still awaiting approval or has been disabled, instead of a generic error, and a finished registration immediately tells you if an administrator's approval is required first

## v0.6.1 — 23 Sep 2026

### Features
- Price freshness now lives in the app header on every page: the "Prices as of" stamp is always visible and clickable to refresh quotes on demand, and data-quality warnings (rate-limited or failed updates, holdings missing an FX rate) follow you as a strip under the header instead of sitting on the dashboard

### Fixes
- The whole interface now follows the chosen language: the dashboard, the portfolio detail (Overview/Positions/Activity and their tables), the portfolios and assets lists, the asset detail tabs, every modal (create portfolio/asset, import portfolio, add transaction, exposure editing), Settings, the Price Sync Health page, and the asset type/class/price-source labels all switch between Italian and English
- On phones the asset header's ⋯ menu is gone: its actions (update from Yahoo, backfill the price history, delete) were already duplicated in the Data tab's danger zone, and the menu kept ending up off-screen when the header wrapped
- On phones the portfolio header's ⋯ menu stays right-aligned when the header wraps, and its panel no longer runs off the screen
- On phones the allocation drill-down no longer closes as soon as it opens: the bottom sheet stays up so you can read the assets behind the slice
- The search panel (⌘K/Ctrl+K, or the header search button) no longer gets stuck after the first use: on a vault with no portfolios or assets it now opens every time instead of working only once
- On touch devices the tooltip of the allocation charts no longer stays pinned on screen: it disappears when you open the drill-down panel or tap outside the chart

## v0.6.0 — 21 Sep 2026

### Features
- Click any allocation slice or bar — the asset-class donut or the sector, region and country bars, on the dashboard and on a portfolio's Allocation tab — to open a side panel (a bottom sheet on phones) listing the assets that make it up, with their value, contribution and share of that slice
- Every chart can be switched to a table view of the same data with the new Chart/Table control on the card — the numbers you were hovering on the chart, now readable row by row, on any device and with a screen reader
- Open the new command palette with ⌘K/Ctrl+K (or the search button in the header, on every device): jump to any page, portfolio or asset, search Yahoo, and run quick actions — add transaction, refresh prices, switch theme, toggle the color-blind palette — all without leaving where you are
- Optional colour-blind-friendly gain/loss palette: from Settings → Preferences you can swap the green/red P/L colours for a blue/orange pair everywhere, text and charts included — the +/− signs and ▲▼ arrows are always shown, and your theme stays untouched
- The portfolio Activity tab can now be filtered by transaction type, asset and date range — the filters stay in the link, so you can share or bookmark a filtered view and the back button brings it back exactly as it was
- Deleting a transaction can now be undone straight from the confirmation toast (5 seconds); the delete-confirmation dialog is gone
- On phones the transaction form opens as a bottom sheet, with the same fields as the desktop dialog
- The asset page is now split into Overview, Exposure and Data tabs, each with its own link you can share or bookmark; the ticker, its identity chips and the latest quote stay pinned at the top while you switch tabs
- The asset Overview now shows "Where held": which of your portfolios hold the asset, with quantity, cost, value and gain/loss — each portfolio links straight to its page
- Asset actions (update from Yahoo, backfill the full price history, delete) moved into the ⋯ menu in the asset header, and are repeated in the Data tab's danger zone
- The portfolio page is now split into Overview, Positions, Activity and Allocation tabs, each with its own link you can share or bookmark; a compact value + P/L strip stays pinned at the top while you switch tabs
- Portfolio actions (export, import, delete) moved into the ⋯ menu in the portfolio header, next to the always-visible "Add transaction" button
- Each portfolio card on the dashboard now shows a small chart of its value trend alongside the current figures
- The dashboard now opens with your net value as a single headline figure, a value-vs-invested chart with 1Y/3Y/ALL ranges, a breakdown you can expand on demand, a "prices as of" freshness stamp, and a data-quality note when something needs attention (like holdings excluded for a missing exchange rate)
- On a fresh vault the dashboard shows a guided first-run checklist: create a portfolio, add an asset, record a transaction — each step links straight to the right page
- You can now switch between the whole vault and a single portfolio from the dashboard header
- New adaptive navigation: on phones a bottom bar (Overview, Portfolios, Assets, More) plus a ⊕ button with quick actions (add transaction, add asset, refresh prices), on tablets a slim icon rail, and on desktop the familiar expandable sidebar — with a compact header that shrinks while you scroll
- The price-sync health page is now called Data & Sync
- The app now follows your system theme by default: light and dark are both first-class, and you can still pin your preferred one from the header
- New interface font (Inter) plus a dedicated monospace font for tickers and codes — both bundled with the app, nothing is downloaded from third-party services
- New Settings → Preferences page: choose theme (Light/Dark/System) and interface language (Italian/English); the app now defaults to the system theme and to Italian, and remembers your choices on this device

### Fixes
- Fetching an asset's exposure from Morningstar now also works for funds quoted on multiple markets: if one market's quotation carries no data, the next quotation of the same ISIN is used automatically

## v0.5.0 — 17 Sep 2026

### Features
- The portfolio transactions list is now paginated: it comes back page by page (20 at a time by default, up to 100) in a stable newest-first order, together with the total count, so long histories stay fast and predictable
- The portfolio detail now has its own monthly/annual percentage performance chart: the same time-weighted return the dashboard shows, measured for that single portfolio in its own currency, with the invested-versus-value capital series next to it
- The portfolio detail allocation section now mirrors the dashboard's "Allocazione complessiva" card: an asset-class donut plus descending region, sector and country bars, all in the portfolio's currency (equity-only, ordered by value)
- The portfolio detail page now shows the same active vs closed investments breakdown as the dashboard, in the portfolio's own currency: open positions with invested, current value, gain/loss and dividends, and the closed part with the cost of the sold lots, the proceeds and the realized gain/loss
- The dashboard now shows a single invested-assets table: one row per asset aggregated across all your portfolios, in your base currency, with invested amount, current value and P/L %, ordered by value
- Choose your base currency in Settings → Profile: the dashboard now consolidates your totals and allocations into that currency
- The dashboard allocation now breaks your whole vault down also by asset class (donut data) and by individual country, alongside the existing sector and macro-region views — all aggregated across your portfolios and expressed in your base currency
- The dashboard now separates active and closed investments at vault and per-portfolio level: for the closed part you see the cost of the sold lots, the proceeds and the realized capital gain/loss; dividend rows are shown with the active investments and, once a position is fully closed, they are included in the proceeds
- New aggregate performance chart on the dashboard: one chart for all your portfolios, in your base currency, showing the percentage time-weighted return of each month or year (with a cumulative TWR line) next to a capital chart of the money invested versus the current value, switchable between Monthly and Annual views. The return is measured day by day, so deposits, sales, dividends and fees no longer distort it and fully closing or reopening a position never produces absurd spikes; positions without a market price are carried at cost and never show up as a fake loss

### Fixes
- Importing a portfolio exported by an older version of the app no longer fails: missing information is filled with sensible defaults
- Portfolios with only closed positions no longer show a misleading -100% gain/loss: closed positions are kept out of the active figures and the amounts no longer carry rounding residues

## v0.4.0 — 13 Sep 2026

### Features
- New dark theme, enabled by default, with Light / Dark / System options
- Redesigned navigation: collapsible sidebar, top header with the theme selector and the user menu, and a slide-in drawer on mobile
- Consistent theme-aware colors across every page and chart, so the app is readable in both light and dark mode
- Destructive actions now use an in-app confirmation dialog instead of the browser's native prompt
- Notifications (toasts) restyled to match the theme and made accessible to screen readers
- Creating assets/portfolios and importing a portfolio now happen in modal dialogs consistent with the app design
- Redesigned sign-in / registration screen with the Peculium logo, a Sign in / Register switch, inline field validation and a password confirmation on registration
- Dashboard rebuilt: KPI cards per currency, an allocation donut and clickable portfolio cards
- The dashboard header shows when prices were last updated
- Portfolio detail: adding/editing a transaction now happens in a modal with inline validation and a live total, and deletion is confirmed in-app
- Portfolio positions and transactions use the shared design-system tables (positions now also show the latest price), and the page actions sit in a sticky header
- Settings reorganized into tabs: Profile, Password, Currencies and Health
- Managed currencies are now picked from a list, with the name filled in automatically
- Changing your password now validates inline and highlights the field at fault (e.g. wrong current password)
- The price-sync health events list is now paginated, so the full history can be browsed
- The price-sync health page moved to a dedicated Admin section in the sidebar

### Fixes
- Portfolio allocations update immediately after adding, editing or deleting a transaction, with no page reload
- Dashboard portfolio history chart draws every portfolio as a continuous line over a real timeline, and can be zoomed and panned like the portfolio charts
- Price-sync health: the Success Rate and Rate Limited cards now show the real values (the rate could stay stuck on `N/A`, or show `NaN%` when there was no data)
- Price-sync health: failed sync messages now state the request type (chart / spark / search / fx) and the related ticker or currency
- Assets not priced by Yahoo (manual / none) no longer generate sync errors in the price-sync health dashboard
- Price-sync health: the summary totals no longer reset on restart and can be scoped to Today / Last 24h / Last 100 events

## v0.3.0 — 11 Sep 2026

### Features
- Choose how each asset gets its prices: `Yahoo`, `Manual` or `None` (avoids Yahoo errors for non-Yahoo tickers such as some bonds)
- The asset price chart now loads the full history and zooms in place using the 1M/3M/1Y/YTD/MAX selectors (no unnecessary reloads)
- New `YTD` (year-to-date) range on the asset price chart
- Stock split markers shown on the asset price chart (e.g. `Split 4:1`)
- Asset exposure (regions/sectors) is edited in a dedicated modal with validated weight tables and one-click fill from JustETF and Yahoo
- Edit the geographic distribution with a per-country list (add/remove countries and set each weight), alongside the regions and sectors
- Geographic and sector exposure can also be filled from Morningstar (official regions), alongside JustETF and Yahoo
- Every distribution shows where its data comes from and when it was last updated (e.g. `from Morningstar (2026-09-05)`)
- Provider lookups are cached, so opening the prefill again is immediate

### Fixes
- Reopening an exposure editor now always starts from the saved data: unsaved changes are discarded

## v0.2.0 — 30 Aug 2026

### Features
- Consistent values in the portfolio summary even when an exchange rate is missing
- Geography and sector distribution charts for portfolios and the dashboard
- Asset classes and allocation by investment class
- Historical exchange rates, so series and charts stay correct over time
- Asset detail page with references, exposure and full price history
- Automatic ETF exposure (countries/regions and sectors) and ticker-to-ISIN lookup

## v0.1.0 — 25 Aug 2026 (first official release)

### Features
- Multi-user registration and login (JWT access + rotating refresh tokens)
- Account settings: edit your name and email, and change your password
- Portfolios: create, edit, delete, export and import
- Assets: create, edit, delete with ticker autocomplete and automatic Yahoo price sync
- Transactions (buy / sell / dividend / split / fee)
- Dashboard: total value, gain/loss, allocation, performance and per-asset ROI
- Portfolio history chart on the dashboard showing how the portfolio value changed over time
- Performance history chart per portfolio with invested amount (cost basis), current value of the still-invested assets and historical realized gain/loss
- Portfolios list with the current value of every portfolio/asset
- Multi-currency support (EUR / USD / GBP / CHF) with a configurable currency whitelist
- Per-currency invested amounts on the dashboard
- Automatic and manual price updates with a price-sync health dashboard
- Automatic periodic refresh of asset prices and exchange rates, updating values and the history up to the latest run
