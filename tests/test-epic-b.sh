#!/usr/bin/env bash
#
# test-epic-b.sh — Manual verification of the geographic/sector allocation
# endpoints (EPIC B.6/B.7).
#
# It walks the whole flow: register/login, ETF asset creation (with the idempotent
# reuse of the existing id on 409), ETF exposure fetch preview (soft-fail) followed
# by an explicit PUT to persist it, reading the persisted exposure, portfolio +
# buys, then the class/geography/sector allocations with reconciliation checks
# (sum of weights ~ 100, class cost basis).
#
# Preconditions:
#   - jq is installed; curl available; the test postgres container reachable (only
#     needed for the optional price seed, not fatal if unavailable).
#
# Test stack lifecycle (isolated, project peculium-test, DB peculium_test, port 8081):
#   Start (build + boot):
#     docker compose -p peculium-test -f docker-compose.test.yml up -d --build python-service backend
#   Wait until the backend answers:
#     until curl -s -o /dev/null http://localhost:8081/api/v1/health/prices; do sleep 1; done
#   Stop and reset (delete the test DB volume):
#     docker compose -p peculium-test -f docker-compose.test.yml down -v
#   NEVER point this at the dev/prod stack (port 8080): it holds real data.
#
# Usage:
#   ./tests/test-epic-b.sh [--step] [--no-seed] [BASE_URL]
#
#   --step     pause after each phase (press Enter to continue)
#   --no-seed  skip the price seed entirely
#   BASE_URL   default http://localhost:8081
#
# Exit code: non-zero if any FAIL was recorded.
#
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

BASE_URL="http://localhost:8081"
API="$BASE_URL/api/v1"
STEP=0
SEED=1

for arg in "$@"; do
  case "$arg" in
    --step) STEP=1 ;;
    --no-seed) SEED=0 ;;
    http://*|https://*) BASE_URL="$arg" ;;
    *) ;;
  esac
done

PASS=0
FAIL=0
WARN=0

note()  { printf '\n\033[1m==> %s\033[0m\n' "$1"; }
ok()    { printf '  \033[32mPASS\033[0m  %s\n' "$1"; PASS=$((PASS+1)); }
bad()   { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; FAIL=$((FAIL+1)); }
warn()  { printf '  \033[33mWARN\033[0m  %s\n' "$1"; WARN=$((WARN+1)); }
die()   { printf '\n\033[31mERRORE: %s\033[0m\n' "$1"; exit 1; }

pause() { [ "$STEP" -eq 1 ] && read -r -p "  Premi Invio per continuare..."; }

# --- guarda: solo stack test isolato (porta 8081) ----------------------------
_PORT="$(printf '%s' "$BASE_URL" | sed -E 's#^.*:([0-9]+)$#\1#')"
_HOST="$(printf '%s' "$BASE_URL" | sed -E 's#^https?://([^:/]+).*#\1#')"
case "$_HOST" in
  localhost|127.0.0.1|0.0.0.0|::1) ;;
  *) warn "host '$BASE_URL' non riconosciuto come local — assicurati di usare lo stack test" ;;
esac
if [ "$_PORT" = "8080" ]; then
  die "BASE_URL punta alla porta 8080 (stack dev/prod con dati reali). Usa lo stack test (porta 8081)! BASE_URL=$BASE_URL"
fi
if [ "$_PORT" != "8081" ]; then
  warn "porta != 8081 ($_PORT): assicurati di puntare allo stack TEST isolato"
fi

printf '\n\033[1mPeculium — EPIC B.6/B.7: allocazione geografica e settoriale\033[0m\n'
printf 'Base URL: %s\n' "$BASE_URL"
printf 'Bersaglio: solo stack TEST isolato (peculium-test, porta 8081).\n\n'

# --- prerequisiti --------------------------------------------------------------
command -v jq >/dev/null 2>&1 || die "jq non installato"
command -v curl >/dev/null 2>&1 || die "curl non installato"

# --- helper login/register ------------------------------------------------------
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

# --- FASE 0: health backend ------------------------------------------------------
note "FASE 0 — Health backend (test stack)"
HC=$(curl -s -o /dev/null -w '%{http_code}' "$API/health" || true)
# fallback probes
if [ "$HC" != "200" ]; then
  HC=$(curl -s -o /dev/null -w '%{http_code}' "$API/health/prices" || true)
fi
# 200/404 = public health route; 401 = auth-protected health route (backend up).
if [ "$HC" = "200" ] || [ "$HC" = "401" ] || [ "$HC" = "404" ]; then
  ok "backend raggiungibile (http $HC)"
else
  warn "health backend ha risposto $HC ($BASE_URL) — verifica che lo stack test sia su"
fi
pause

# --- FASE 1: register/login -------------------------------------------------------
note "FASE 1 — Registrazione e login utente"
EMAIL="epicb@test.local"
PW="Password123!"
register "$EMAIL" "$PW" "Epic B"
TOK=$(login "$EMAIL" "$PW")
if [ -n "$TOK" ]; then
  ok "login riuscito ($EMAIL)"
else
  die "login fallito — impossibile proseguire"
fi
pause

# --- FASE 2: creazione asset ETF (SMEA.MI) -----------------------------------------
note "FASE 2 — Crea asset ETF SMEA.MI (idempotente al rerun)"
CREATE1=$(curl -s -w '\n%{http_code}' -X POST "$API/assets" \
  -H "Authorization: Bearer $TOK" \
  -H 'Content-Type: application/json' \
  -d '{"ticker":"SMEA.MI","name":"iShares Core MSCI Europe UCITS ETF EUR (Acc)","type":"etf","currency":"EUR","asset_class":"equity"}')
BODY1="${CREATE1%$'\n'*}"
CODE1="${CREATE1##*$'\n'}"
AI1=$(jq -r '.id // empty' <<<"$BODY1")
if [ "$CODE1" = "201" ]; then
  ok "asset SMEA.MI creato (201, id=$AI1)"
elif [ "$CODE1" = "409" ] && [ -n "$AI1" ]; then
  ok "asset SMEA.MI gia esistente (409) — riuso id esistente ($AI1)"
else
  bad "creazione SMEA.MI -> http $CODE1"
fi
[ -n "$AI1" ] || die "impossibile determinare l'id per SMEA.MI"

# duplicate create per SMEA.MI -> atteso 409
DUP1=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/assets" \
  -H "Authorization: Bearer $TOK" \
  -H 'Content-Type: application/json' \
  -d '{"ticker":"SMEA.MI","name":"iShares Core MSCI Europe UCITS ETF EUR (Acc)","type":"etf","currency":"EUR","asset_class":"equity"}')
[ "$DUP1" = "409" ] && ok "duplicato SMEA.MI -> 409 (atteso)" || bad "duplicato SMEA.MI -> $DUP1 (atteso 409)"

# --- FASE 3: creazione asset ETF (SXR8.DE) -----------------------------------------
note "FASE 3 — Crea asset ETF SXR8.DE (idempotente al rerun)"
CREATE2=$(curl -s -w '\n%{http_code}' -X POST "$API/assets" \
  -H "Authorization: Bearer $TOK" \
  -H 'Content-Type: application/json' \
  -d '{"ticker":"SXR8.DE","name":"iShares Core S&P 500 UCITS ETF EUR (Acc)","type":"etf","currency":"EUR","asset_class":"equity"}')
BODY2="${CREATE2%$'\n'*}"
CODE2="${CREATE2##*$'\n'}"
AI2=$(jq -r '.id // empty' <<<"$BODY2")
if [ "$CODE2" = "201" ]; then
  ok "asset SXR8.DE creato (201, id=$AI2)"
elif [ "$CODE2" = "409" ] && [ -n "$AI2" ]; then
  ok "asset SXR8.DE gia esistente (409) — riuso id esistente ($AI2)"
else
  bad "creazione SXR8.DE -> http $CODE2"
fi
[ -n "$AI2" ] || die "impossibile determinare l'id per SXR8.DE"

DUP2=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/assets" \
  -H "Authorization: Bearer $TOK" \
  -H 'Content-Type: application/json' \
  -d '{"ticker":"SXR8.DE","name":"iShares Core S&P 500 UCITS ETF EUR (Acc)","type":"etf","currency":"EUR","asset_class":"equity"}')
[ "$DUP2" = "409" ] && ok "duplicato SXR8.DE -> 409 (atteso)" || bad "duplicato SXR8.DE -> $DUP2 (atteso 409)"
pause

# --- FASE 4: fetch exposure ETF (preview) + PUT di persistenza (SOFT-FAIL) ---------
note "FASE 4 — Fetch ETF exposure (preview non persistente) + PUT exposure (richiede python-service + internet JustETF; soft-fail)"
for spec in "$AI1:SMEA.MI" "$AI2:SXR8.DE"; do
  aid="${spec%%:*}"; tkr="${spec##*:}"
  RES=$(curl -s -w '\n%{http_code}' -X POST "$API/assets/$aid/fetch-etf-exposure" \
    -H "Authorization: Bearer $TOK")
  BODY="${RES%$'\n'*}"; CODE="${RES##*$'\n'}"
  if [ "$CODE" = "200" ] || [ "$CODE" = "201" ]; then
    printf '    %s exposure fetch ok (http %s, preview NON persistita)\n' "$tkr" "$CODE"
    jq . <<<"$BODY" 2>/dev/null || printf '    raw: %s\n' "$BODY"
    ok "exposure preview fetched per $tkr"
    # Il fetch e' una preview read-only (solo l'ISIN viene persistito): la
    # persistenza avviene SOLO con il PUT esplicito.
    PUT_BODY=$(jq -c '{countries, regions, sectors}' <<<"$BODY" 2>/dev/null)
    if [ -n "$PUT_BODY" ] && [ "$PUT_BODY" != "null" ]; then
      PUT_RES=$(curl -s -w '\n%{http_code}' -X PUT "$API/assets/$aid/exposure" \
        -H "Authorization: Bearer $TOK" \
        -H 'Content-Type: application/json' \
        -d "$PUT_BODY")
      PUT_BODY_RES="${PUT_RES%$'\n'*}"; PUT_CODE="${PUT_RES##*$'\n'}"
      if [ "$PUT_CODE" = "200" ]; then
        ok "exposure salvata con PUT per $tkr (http $PUT_CODE)"
      else
        warn "$tkr: PUT exposure -> http $PUT_CODE (${PUT_BODY_RES}) — exposure non persistita, allocazioni 0 per questo asset"
      fi
    else
      warn "$tkr: risposta fetch senza dimensions estraibili (jq) — PUT saltato, allocazioni resteranno 0 per questo asset"
    fi
  else
    warn "$tkr: fetch-etf-exposure -> http $CODE (python-service/internet non disponibili?) — allocazioni resteranno 0 per questo asset"
  fi
done

# --- FASE 5: lettura exposure persistita (dal PUT della FASE 4) ----------------------
note "FASE 5 — Lettura exposure persistita (GET /assets/{id}/exposure, salvata con PUT in FASE 4)"
for spec in "$AI1:SMEA.MI" "$AI2:SXR8.DE"; do
  aid="${spec%%:*}"; tkr="${spec##*:}"
  EXP=$(curl -s "$API/assets/$aid/exposure" -H "Authorization: Bearer $TOK")
  # NOTE: `jq -e` prints the boolean to stdout, so discard stdout and keep the
  # exit status only (broken as `HAS_REG=$(jq -e ... && echo ...)` before).
  HAS_REG=$(jq -e 'has("regions")' >/dev/null <<<"$EXP" 2>/dev/null && echo yes || echo no)
  HAS_SEC=$(jq -e 'has("sectors")' >/dev/null <<<"$EXP" 2>/dev/null && echo yes || echo no)
  if [ "$HAS_REG" = "yes" ] && [ "$HAS_SEC" = "yes" ]; then
    ok "$tkr: exposure contiene regions e sectors"
    printf '    regions (%s):\n' "$(jq '.regions | length' <<<"$EXP")"
    jq '.regions[] | "\(.name): \(.weight)"' <<<"$EXP" 2>/dev/null | head -n 12
    printf '    sectors (%s):\n' "$(jq '.sectors | length' <<<"$EXP")"
    jq '.sectors[] | "\(.name): \(.weight)"' <<<"$EXP" 2>/dev/null | head -n 12
  else
    warn "$tkr: exposure mancante di regions/sectors (atteso 0 dopo soft-fail)"
  fi
done
pause

# --- FASE 6: seed prezzi (opzionale, non fatale) -------------------------------------
if [ "$SEED" -eq 1 ]; then
  note "FASE 6 — Seed prezzi manuale (solo stack test, NON fatale)"
  if [ "$_PORT" = "8081" ]; then
    SEED_CMD=(docker compose -p peculium-test -f "$SCRIPT_DIR/../docker-compose.test.yml" exec -T postgres psql -U peculium -d peculium_test)
    if ! docker ps >/dev/null 2>&1; then
      warn "docker non disponibile — prices not seeded, weights will be 0"
    else
      if "${SEED_CMD[@]}" < "$SCRIPT_DIR/seed-prices.sql" >/dev/null 2>&1; then
        ok "seed prezzi applicato (SMEA.MI @105, SXR8.DE @310)"
      else
        warn "seed prezzi fallito (container postgres test non attivo?) — prices not seeded, weights will be 0"
      fi
    fi
  else
    warn "seed skipped — BASE_URL non sulla porta 8081 (non stack test)"
  fi
else
  note "FASE 6 — Seed prezzi saltato (--no-seed)"
fi
pause

# --- FASE 7: creazione portafoglio ----------------------------------------------------
note "FASE 7 — Creazione portafoglio (EUR, idempotente al rerun)"
EXISTING_PF=$(curl -s "$API/portfolios" -H "Authorization: Bearer $TOK")
# A user with no portfolios gets a `null` body (nil Go slice) — never iterate on it.
PID=$(jq -r 'if type == "array" then map(select(.name == "BitPie portfolio"))[0].id // empty else empty end' <<<"$EXISTING_PF")
if [ -n "$PID" ]; then
  ok "portafoglio esistente riusato ($PID)"
else
  PF=$(curl -s -X POST "$API/portfolios" \
    -H "Authorization: Bearer $TOK" \
    -H 'Content-Type: application/json' \
    -d '{"name":"BitPie portfolio","description":"EPIC B manual test","currency":"EUR"}')
  PID=$(jq -r '.id // empty' <<<"$PF")
  if [ -n "$PID" ]; then
    ok "portafoglio creato ($PID)"
  else
    die "creazione portafoglio fallita: $PF"
  fi
fi
pause

# --- FASE 8: transazioni buy (salta se gia presenti al rerun) ----------------------------
note "FASE 8 — Transazioni buy (salta se gia presenti al rerun)"
TXS=$(curl -s "$API/portfolios/$PID/transactions" -H "Authorization: Bearer $TOK")
if [ "$(jq '.total' <<<"$TXS")" -gt 0 ]; then
  note "(rerun) transazioni gia presenti — creazione buy saltata per evitare accumulo"
  ok "rilevate transazioni esistenti, buy saltate"
else
  BUY1=$(curl -s -w '\n%{http_code}' -X POST "$API/portfolios/$PID/transactions" \
    -H "Authorization: Bearer $TOK" \
    -H 'Content-Type: application/json' \
    -d "{\"asset_id\":\"$AI1\",\"type\":\"buy\",\"quantity\":10,\"price\":100.00,\"date\":\"2024-01-15T00:00:00Z\"}")
  C1="${BUY1##*$'\n'}"
  [ "$C1" = "201" ] && ok "buy SMEA.MI (10 @ 100 EUR) -> 201" || bad "buy SMEA.MI -> $C1 (atteso 201)"

  BUY2=$(curl -s -w '\n%{http_code}' -X POST "$API/portfolios/$PID/transactions" \
    -H "Authorization: Bearer $TOK" \
    -H 'Content-Type: application/json' \
    -d "{\"asset_id\":\"$AI2\",\"type\":\"buy\",\"quantity\":5,\"price\":300.00,\"date\":\"2024-01-15T00:00:00Z\"}")
  C2="${BUY2##*$'\n'}"
  [ "$C2" = "201" ] && ok "buy SXR8.DE (5 @ 300 EUR) -> 201" || bad "buy SXR8.DE -> $C2 (atteso 201)"
fi
pause

# --- FASE 9: class allocation (baseline per la riconciliazione) -------------------------
note "FASE 9 — Allocazione per classe (baseline, cost basis 2500 EUR)"
CLASS=$(curl -s "$API/portfolios/$PID/allocation/class" -H "Authorization: Bearer $TOK")
jq . <<<"$CLASS" 2>/dev/null || printf 'raw: %s\n' "$CLASS"
# JSON decimals are strings: convert with tonumber before adding (plain add would concatenate).
# Expected total: 2500 = cost basis (10*100 + 5*300) without prices, or
# 2600 = current value with the seed (10*105 + 5*310).
VALSUM=$(jq '[.classes[]?.value // "0" | tonumber? // 0] | add // 0' <<<"$CLASS")
if awk -v v="$VALSUM" 'BEGIN{ ok = ((v >= 2500*0.985 && v <= 2500*1.015) || (v >= 2600*0.985 && v <= 2600*1.015)); exit !ok }'; then
  ok "somma class values ~ 2500/2600 EUR (got $VALSUM)"
else
  warn "somma class values = $VALSUM (atteso ~2500 senza prox / ~2600 con seed 105-310)"
fi
TOTW=$(jq '[.classes[]?.weight // "0" | tonumber? // 0] | add // 0' <<<"$CLASS")
awk -v w="$TOTW" 'BEGIN{ ok = (w >= 99 && w <= 101); exit !ok }' \
  && ok "class: somma pesi ~ 100 (got $TOTW)" || warn "class: somma pesi = $TOTW (non ~100)"
pause

# --- FASE 10: allocazione geografica ------------------------------------------------
note "FASE 10 — Allocazione geografica (geography)"
GEO=$(curl -s "$API/portfolios/$PID/allocation/geography" -H "Authorization: Bearer $TOK")
jq . <<<"$GEO" 2>/dev/null || printf 'raw: %s\n' "$GEO"
# struttura: array presente, ogni voce con region+value+weight
if jq -e '.regions | type == "array"' <<<"$GEO" >/dev/null 2>&1 \
   && jq -e 'all(.regions[]; (has("region") and has("value") and has("weight")))' <<<"$GEO" >/dev/null 2>&1; then
  ok "geography: array presente, ogni voce ha region+value+weight"
else
  bad "geography: struttura inattesa"
fi
GW=$(jq '[.regions[]?.weight // "0" | tonumber? // 0] | add // 0' <<<"$GEO")
awk -v w="$GW" 'BEGIN{ ok = (w >= 98 && w <= 101); exit !ok }' \
  && ok "geography: somma pesi ~ 100 (got $GW)" || warn "geography: somma pesi = $GW (non ~100, probabilmente zero senza seed)"
pause

# --- FASE 11: allocazione settoriale ------------------------------------------------
note "FASE 11 — Allocazione settoriale (sector)"
SEC=$(curl -s "$API/portfolios/$PID/allocation/sector" -H "Authorization: Bearer $TOK")
jq . <<<"$SEC" 2>/dev/null || printf 'raw: %s\n' "$SEC"
if jq -e '.sectors | type == "array"' <<<"$SEC" >/dev/null 2>&1 \
   && jq -e 'all(.sectors[]; (has("sector") and has("value") and has("weight")))' <<<"$SEC" >/dev/null 2>&1; then
  ok "sector: array presente, ogni voce ha sector+value+weight"
else
  bad "sector: struttura inattesa"
fi
SW=$(jq '[.sectors[]?.weight // "0" | tonumber? // 0] | add // 0' <<<"$SEC")
# Real JustETF sector tables do not cover the whole fund (the remainder stays in
# the "Other / Not Classified" row of the REGIONS dimension), so the sector
# weights typically sum to ~97-98 instead of 100. Accept >= 85 as sane.
awk -v w="$SW" 'BEGIN{ ok = (w >= 85 && w <= 101); exit !ok }' \
  && ok "sector: somma pesi = $SW (ok: >=85, quota non classificata esclusa)" \
  || warn "sector: somma pesi = $SW (atteso ~97-100 con seed, 0 senza)"
pause

# --- riepilogo -----------------------------------------------------------------------
printf '\n\033[1mRisultato: %d PASS, %d FAIL (%d warning)\033[0m\n' "$PASS" "$FAIL" "$WARN"
exit "$([ "$FAIL" -eq 0 ] && echo 0 || echo 1)"
