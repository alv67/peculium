# Workflow preferences

- **Language**: GitHub issues, commits, PRs, and code comments must always be in
  **English** (titles and bodies), regardless of the language used in chat.
- Always propose the solution and discuss it with the user BEFORE writing any
  code. Do not implement a chosen technical direction on your own initiative.
- After changes, restart the stack with `make down` and `make up` to test
  (force-recreating containers is not always sufficient).
- Run e2e/smoke tests ONLY against the isolated test stack: `make test-e2e`
  (`docker-compose.test.yml`, project `peculium-test`, separate DB `peculium_test`,
  ports 8081/5433/6380, Yahoo finance disabled). NEVER test against the dev stack
  (`docker-compose.dev.yml` / `make up`, DB `peculium`, port 8080) or the release
  stack (`docker-compose.yml`): they hold real data and must stay clean.
- **Never run schema migrations on the shared dev stack**: a change that adds or
  edits a migration must be exercised on the isolated test stack
  (`make test-e2e` / `docker-compose.test.yml`), never via `make up` on the dev
  DB. Migrating the real dev DB ahead of `develop` leaves the dev backend
  crash-looping on the migration-version mismatch until the code catches up.
- Delegate implementation to the dedicated subagents whenever the work fits
  their scope: `backend` for Go/Postgres/Redis/API, `frontend` for SvelteKit/
  TypeScript/Tailwind, `python` for the `python-service/` ETF metadata microservice
  (FastAPI/JustETF/Morningstar), `finanza` for financial/statistical analysis. Use
  `explore`/`general` for research or cross-cutting tasks. Do not implement
  domain work yourself when a dedicated subagent exists.
- For manual API testing use the isolated test stack and `tests/api-test.http`
  (VS Code REST Client), never the dev/prod stack. The EPIC B allocation smoke
  test is `tests/test-epic-b.sh` on the same isolated stack (seeds prices via
  `tests/seed-prices.sql`, since Yahoo is disabled there). The EPIC K
  transaction-filter and allocation drill-down smoke test is
  `tests/test-epic-k.sh` (same isolated stack and price seed).
- **Issue triage**: external issues opened on the public repo arrive through the
  issue forms in `.github/ISSUE_TEMPLATE/` with the `triage` label applied
  automatically. The maintainers' planning issues never use `triage`; they carry
  `epic:*`/`priority` labels. Treat `label:triage` as the incoming queue and
  `label:epic:*` as the project plan.
- **Keep the project docs in sync before closing a PR**: always check the
  project documents first (AGENTS.md, PLAN.md, STATUS.md, `docs/` guides en/it)
  and update them together with the code. A PR must NOT be closed until its
  related documentation (endpoints, behavior, UI, status tables) is updated.
- **Screenshots for docs/website**: capture every UI screenshot in **both
  locales** (Italian and English), from the isolated test stack, and embed the
  one that matches the language of the page that uses it.
- **The guides are a snapshot of the current version**: `docs/FRONTEND-GUIDE`,
  `docs/BACKEND-GUIDE` and `docs/DATABASE-GUIDE` (both `.en.md` and `.it.md`)
  describe **how the app works right now**, as if written against the latest
  commit. Write in the present tense and keep the two languages mirror images
  (same sections, same facts). Never add development history: no references to
  issues/PRs, epics or phases, design-decision IDs, spec sections, release
  versions, or change wording ("now", "no longer", "previously", "was
  replaced/renamed/moved", "fast-follow", "coming soon", "Notes and open
  points"). When a feature changes, rewrite the affected sentences so they state
  the new behaviour directly — do not narrate the change. `docs/UX-REDESIGN.*.md`
  is the separate redesign specification and is exempt from this rule.
- **Release notes** (`docs/RELEASE-NOTES.en.md` / `.it.md`): one line per
  feature/fix, written from the end user's point of view (what they see and use
  in the app — no internal/backend details). Add the lines under an `Unreleased`
  section on top; only when there are pending changes. When publishing, rename
  that section to `## <version> — <date>` (do NOT leave an empty `Unreleased`).
  Sections are ordered **newest first**: the latest released version on top,
  down to the oldest. Group the lines under `Features` and `Fixes` (in Italian,
  `Nuove funzionalità` / `Correzioni`); omit `Fixes` when a release has none.
  Include only what the end user perceives as different from the **previous
  release**: never list bugs that were introduced and fixed within the same
  unreleased development cycle (users never saw them, so they are not release
  notes — they are just development activity).
