#!/usr/bin/env bash
#
# test-fx-history.sh — Manual verification of the per-date FX rate history
# (EPIC B.9): prices change nothing, but a material EUR/USD move between two
# dates must distort the USD portfolio history accordingly.
#
# Scenario:
#   - a portfolio denominated in USD holding 1 unit of an EUR asset (VLABEUR1,
#     an intentionally unknown ticker so Yahoo fetching soft-fails and never
#     overwrites the seeded prices; only the recommended smoke trigger is used:
#     seed via SQL + no-op PATCH /portfolios/{id} to recompute the series).
#   - the asset has a STABLE price of 100 EUR on 2024-01-15 and 2024-03-15;
#   - USD->EUR history: 0.90 on 2024-01-15, 1.10 on 2024-03-15 (seeded via SQL).
#   - Expected market values (USD): 100/0.90 ≈ 111.11 on 2024-01-15, and
#     100/1.10 ≈ 90.91 on 2024-03-15 — i.e. price unchanged, value distorted by
#     the FX move (ratio 0.90/1.10 ≈ 0.818).
#
# Preconditions:
#   - jq is installed; curl available; a compose tool (docker or podman-compose)
#     is available and the test postgres container is reachable (required for
#     the SQL seed of prices and FX history).
#
# Test stack lifecycle (isolated, project peculium-test, DB peculium_test, port 8081):
#   Start (build + boot):
#     docker compose -p peculium-test -f docker-compose.test.yml up -d --build
#   Wait until the backend answers:
#     until curl -s -o /dev/null http://localhost:8081/api/v1/health/prices; do sleep 1; done
#   Stop and reset (delete the test DB volume):
#     docker compose -p peculium-test -f docker-compose.test.yml down -v
#   NEVER point this at the dev/prod stack (port 8080): it holds real data.
#
# Usage:
#   ./tests/test-fx-history.sh [--step] [BASE_URL]
#
#   --step     pause after each phase (press Enter to continue)
#   BASE_URL   default http://localhost:8081
#
# Exit code: non-zero if any FAIL was recorded.
#
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

BASE_URL="http://localhost:8081"
API="$BASE_URL/api/v1"
STEP=0

for arg in "$@"; do
  case "$arg" in
    --step) STEP=1 ;;
    http://*|https://*) BASE_URL="$arg" ;;
    *) ;;
  esac
done

PASS=0
FAIL=0
WARN=0

# Compose binary detection (mirrors the Makefile): docker > docker-compose > podman-compose.
COMPOSE=""
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
elif command -v podman-compose >/dev/null 2>&1; then
  COMPOSE="podman-compose"
fi
[ -n "$COMPOSE" ] || die "no compose tool found (docker or podman-compose)"

note()  { printf '\n\033[1m==> %s\033[0m\n' "$1"; }
ok()    { printf '  \033[32mPASS\033[0m  %s\n' "$1"; PASS=$((PASS+1)); }
bad()   { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; FAIL=$((FAIL+1)); }
warn()  { printf '  \033[33mWARN\033[0m  %s\n' "$1"; WARN=$((WARN+1)); }
die()   { printf '\n\033[31mERROR: %s\033[0m\n' "$1"; exit 1; }

pause() { [ "$STEP" -eq 1 ] && read -r -p "  Press Enter to continue..."; }

# --- guard: only the isolated test stack (port 8081) --------------------------
_PORT="$(printf '%s' "$BASE_URL" | sed -E 's#^.*:([0-9]+)$#\1#')"
_HOST="$(printf '%s' "$BASE_URL" | sed -E 's#^https?://([^:/]+).*#\1#')"
case "$_HOST" in
  localhost|127.0.0.1|0.0.0.0|::1) ;;
  *) warn "host '$BASE_URL' is not local — make sure you target the test stack" ;;
esac
if [ "$_PORT" = "8080" ]; then
  die "BASE_URL targets port 8080 (dev/prod stack with real data). Use the test stack (port 8081)! BASE_URL=$BASE_URL"
fi
if [ "$_PORT" != "8081" ]; then
  warn "port != 8081 ($_PORT): make sure you target the isolated TEST stack"
fi

printf '\n\033[1mPeculium — EPIC B.9: per-date FX history (series distortion)\033[0m\n'
printf 'Base URL: %s\n' "$BASE_URL"
printf 'Target:  isolated TEST stack only (peculium-test, port 8081).\n\n'

# --- prerequisites --------------------------------------------------------------
command -v jq >/dev/null 2>&1 || die "jq is not installed"
command -v curl >/dev/null 2>&1 || die "curl is not installed"

# --- helper login/register --------------------------------------------------------
login() { # $1 email  $2 password
  local body
  body=$(curl -s -X POST "$API/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$1\",\"password\":\"$2\"}")
  jq -r '.access_token // empty' <<<"$body"
}

register() { # $1 email  $2 password  $3 name
  curl -s -X POST "$API/auth/register" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$1\",\"password\":\"$2\",\"name\":\"$3\"}" >/dev/null
}

# --- FASE 0: backend health --------------------------------------------------------
note "FASE 0 — Backend health (test stack)"
HC=$(curl -s -o /dev/null -w '%{http_code}' "$API/health" || true)
if [ "$HC" != "200" ]; then
  HC=$(curl -s -o /dev/null -w '%{http_code}' "$API/health/prices" || true)
fi
if [ "$HC" = "200" ] || [ "$HC" = "401" ] || [ "$HC" = "404" ]; then
  ok "backend reachable (http $HC)"
else
  warn "backend health answered $HC ($BASE_URL) — verify the test stack is up"
fi
pause

# --- FASE 1: register/login ----------------------------------------------------------
note "FASE 1 — User registration and login"
EMAIL="fxsmoke@test.local"
PW="Password123!"
register "$EMAIL" "$PW" "FX Smoke"
TOK=$(login "$EMAIL" "$PW")
if [ -n "$TOK" ]; then
  ok "login ok ($EMAIL)"
else
  die "login failed — cannot proceed"
fi
pause

# --- FASE 2: create EUR asset (VLABEUR1, unknown to Yahoo on purpose) -----------------
note "FASE 2 — Create EUR asset VLABEUR1 (idempotent on rerun)"
CREATE=$(curl -s -w '\n%{http_code}' -X POST "$API/assets" \
  -H "Authorization: Bearer $TOK" \
  -H 'Content-Type: application/json' \
  -d '{"ticker":"VLABEUR1","name":"Smoke EUR ETF (unknown to Yahoo)","type":"etf","currency":"EUR"}')
BODY="${CREATE%$'\n'*}"
CODE="${CREATE##*$'\n'}"
AID=$(jq -r '.id // empty' <<<"$BODY")
if [ "$CODE" = "201" ]; then
  ok "asset VLABEUR1 created (201, id=$AID)"
elif [ "$CODE" = "409" ] && [ -n "$AID" ]; then
  ok "asset VLABEUR1 already exists (409) — reusing id ($AID)"
else
  bad "asset creation -> http $CODE"
fi
[ -n "$AID" ] || die "cannot determine the asset id for VLABEUR1"
pause

# --- FASE 3: create USD portfolio ------------------------------------------------------
note "FASE 3 — Create USD portfolio (idempotent on rerun)"
EXISTING=$(curl -s "$API/portfolios" -H "Authorization: Bearer $TOK")
PID=$(jq -r 'if type == "array" then map(select(.name == "FX Smoke"))[0].id // empty else empty end' <<<"$EXISTING")
if [ -n "$PID" ]; then
  ok "portfolio reused ($PID)"
else
  PF=$(curl -s -X POST "$API/portfolios" \
    -H "Authorization: Bearer $TOK" \
    -H 'Content-Type: application/json' \
    -d '{"name":"FX Smoke","description":"EPIC B.9 manual test","currency":"USD"}')
  PID=$(jq -r '.id // empty' <<<"$PF")
  if [ -n "$PID" ]; then
    ok "portfolio created ($PID)"
  else
    die "portfolio creation failed: $PF"
  fi
fi
pause

# --- FASE 4: buy transaction (skip if already present on rerun) ---------------------------
note "FASE 4 — Buy 1 unit @ 100 EUR on 2024-01-10 (skip if present on rerun)"
TXS=$(curl -s "$API/portfolios/$PID/transactions" -H "Authorization: Bearer $TOK")
if [ "$(jq '.total' <<<"$TXS")" -gt 0 ]; then
  note "(rerun) transactions already present — buy skipped"
  ok "existing transactions detected, buy skipped"
else
  BUY=$(curl -s -w '\n%{http_code}' -X POST "$API/portfolios/$PID/transactions" \
    -H "Authorization: Bearer $TOK" \
    -H 'Content-Type: application/json' \
    -d "{\"asset_id\":\"$AID\",\"type\":\"buy\",\"quantity\":1,\"price\":100.00,\"date\":\"2024-01-10T00:00:00Z\"}")
  CB="${BUY##*$'\n'}"
  [ "$CB" = "201" ] && ok "buy VLABEUR1 (1 @ 100 EUR) -> 201" || bad "buy VLABEUR1 -> $CB (expected 201)"
fi
pause

# --- FASE 5: SQL seed (prices + FX history) on the test stack -----------------------------
note "FASE 5 — SQL seed: stable EUR price + USD->EUR history (0.90 / 1.10)"
if [ "$_PORT" = "8081" ]; then
  SEED_CMD=(docker compose -p peculium-test -f "$SCRIPT_DIR/../docker-compose.test.yml" exec -T postgres psql -U peculium -d peculium_test)
  if "${SEED_CMD[@]}" < "$SCRIPT_DIR/seed-fx-history.sql" >/dev/null 2>&1; then
    ok "FX history seed applied (price 100 both dates; USD->EUR 0.90 / 1.10)"
  else
    die "FX history seed failed (is the test postgres container up?)"
  fi
else
  die "SQL seed requires the test stack on port 8081 — nothing was seeded"
fi
pause

# --- FASE 6: recompute trigger (no-op PATCH -> series.Recompute) ---------------------------
note "FASE 6 — Trigger series recompute via PATCH /portfolios/{id}"
PATCH=$(curl -s -w '\n%{http_code}' -X PATCH "$API/portfolios/$PID" \
  -H "Authorization: Bearer $TOK" \
  -H 'Content-Type: application/json' \
  -d '{"name":"FX Smoke","description":"EPIC B.9 manual test","currency":"USD"}')
CP="${PATCH##*$'\n'}"
[ "$CP" = "200" ] && ok "PATCH portfolio -> 200 (series recomputed)" || bad "PATCH portfolio -> $CP (expected 200)"
pause

# --- FASE 7: read history and assert per-date FX distortion --------------------------------
note "FASE 7 — GET /portfolios/{id}/history: per-date FX distortion"
HIST=$(curl -s "$API/portfolios/$PID/history" -H "Authorization: Bearer $TOK")
jq . <<<"$HIST" >/dev/null 2>&1 || printf 'raw: %s\n' "$HIST"

D1="2024-01-15"; D2="2024-03-15"
MV1=$(jq --arg d "$D1" '[.series[]? | select(.date | startswith($d))] | if length > 0 then .[0].market_value | tonumber? // 0 else 0 end' <<<"$HIST")
MV2=$(jq --arg d "$D2" '[.series[]? | select(.date | startswith($d))] | if length > 0 then .[0].market_value | tonumber? // 0 else 0 end' <<<"$HIST")

printf '    market value %s: %s\n' "$D1" "$MV1"
printf '    market value %s: %s\n' "$D2" "$MV2"

HAS1=$(jq --arg d "$D1" '[.series[]? | select(.date | startswith($d))] | length' <<<"$HIST")
HAS2=$(jq --arg d "$D2" '[.series[]? | select(.date | startswith($d))] | length' <<<"$HIST")

if [ "$HAS1" -ge 1 ] && [ "$HAS2" -ge 1 ]; then
  ok "both expected dates present in the series"
else
  bad "missing expected dates in the series (d1=$HAS1 d2=$HAS2) — maybe the series is empty?"
fi

# Expected: mv1 = 100/0.90 ≈ 111.11 USD, mv2 = 100/1.10 ≈ 90.91 USD (2% tol).
awk -v v="$MV1" 'BEGIN{ ok = (v >= 108.9 && v <= 113.3 && v > 0); exit !ok }' \
  && ok "mv($D1) ~ 111.11 USD (100 EUR / 0.90) — got $MV1" \
  || bad "mv($D1) = $MV1 (expected ~111.11, 2%% band)"
awk -v v="$MV2" 'BEGIN{ ok = (v >= 89.1 && v <= 92.7 && v > 0); exit !ok }' \
  && ok "mv($D2) ~ 90.91 USD (100 EUR / 1.10) — got $MV2" \
  || bad "mv($D2) = $MV2 (expected ~90.91, 2%% band)"

# Ratio mv2/mv1 must equal 0.90/1.10 ≈ 0.818 (3% band) — the discriminator that
# proves the value moved because of FX, not because the price changed.
R=$(awk -v a="$MV1" -v b="$MV2" 'BEGIN{ if (a > 0) printf "%.6f", b/a; else printf "0" }')
awk -v r="$R" 'BEGIN{ ok = (r >= 0.793 && r <= 0.843); exit !ok }' \
  && ok "ratio mv($D2)/mv($D1) ~ 0.818 (0.90/1.10) — got $R" \
  || bad "ratio mv($D2)/mv($D1) = $R (expected ~0.818) — FM per-date resolution looks broken"

# Asset-level series carries the same per-date values (sanity spot-check).
ASERIES=$(jq -c --arg aid "$AID" '.assets[]? | select(.asset_id == $aid) | .series' <<<"$HIST")
[ -n "$ASERIES" ] && [ "$ASERIES" != "null" ] \
  && ok "asset-level series present for VLABEUR1" || warn "asset-level series missing for VLABEUR1"
pause

# --- summary --------------------------------------------------------------------------------
printf '\n\033[1mResult: %d PASS, %d FAIL (%d warnings)\033[0m\n' "$PASS" "$FAIL" "$WARN"
exit "$([ "$FAIL" -eq 0 ] && echo 0 || echo 1)"