#!/usr/bin/env bash
#
# test-epic-k.sh — Smoke test for the EPIC K backend additions:
#   * K.4c transaction filters (GET /portfolios/{id}/transactions?type&asset_id&from&to)
#   * K.5 allocation drill-down (GET /portfolios/{id}/allocation/drill,
#     GET /dashboard/allocation/drill)
#
# It walks a deterministic flow: register/login, two equity ETFs, synthetic
# exposure persisted with PUT (so the drill buckets are exact and no external
# provider is needed), an optional manual price seed, a portfolio with two buys
# and one dividend, then the filter matrix and the drill-down for every
# dimension (class/country/region/sector) with reconciliation against the
# matching allocation bucket.
#
# Preconditions:
#   - jq installed; curl available; the test postgres container reachable (only
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
#   ./tests/test-epic-k.sh [--step] [--no-seed] [BASE_URL]
#
#   --step     pause after each phase (press Enter to continue)
#   --no-seed  skip the price seed entirely (drill totals then stay 0)
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

# --- guard: solo stack test isolato (porta 8081) ------------------------------
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

printf '\n\033[1mPeculium — EPIC K: filtri transazioni + drill-down allocazioni\033[0m\n'
printf 'Base URL: %s\n' "$BASE_URL"
printf 'Bersaglio: solo stack TEST isolato (peculium-test, porta 8081).\n\n'

# --- prerequisiti -------------------------------------------------------------
command -v jq >/dev/null 2>&1 || die "jq non installato"
command -v curl >/dev/null 2>&1 || die "curl non installato"

# Compose command detection, same order as the Makefile (docker compose,
# docker-compose, podman-compose) — used only for the optional price seed.
COMPOSE="$(if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then echo 'docker compose'; elif command -v docker-compose >/dev/null 2>&1; then echo 'docker-compose'; elif command -v podman-compose >/dev/null 2>&1; then echo 'podman-compose'; else echo 'podman-compose'; fi)"

# --- helper login/register ----------------------------------------------------
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

# num_eq: true when |$1 - $2| <= $3 (tolerance), all numbers.
num_eq() { awk -v a="$1" -v b="$2" -v t="$3" 'BEGIN{ d=a-b; if(d<0)d=-d; exit !(d<=t) }'; }

# urlenc: percent-encode a value (for bucket keys with spaces).
urlenc() { jq -rn --arg v "$1" '$v|@uri'; }

# bucket_value: read a decimal value from an allocation payload (stdin JSON),
# $1 = dimension (class|country|region|sector), $2 = key.
bucket_value() {
  case "$1" in
    class)   jq -r --arg k "$2" '(.classes[]?  | select(.class==$k)   | .value) // empty' ;;
    country) jq -r --arg k "$2" '(.countries[]?| select(.country==$k) | .value) // empty' ;;
    region)  jq -r --arg k "$2" '(.regions[]?  | select(.region==$k)  | .value) // empty' ;;
    sector)  jq -r --arg k "$2" '(.sectors[]?  | select(.sector==$k)  | .value) // empty' ;;
  esac
}

# --- FASE 0: health backend ---------------------------------------------------
note "FASE 0 — Health backend (test stack)"
HC=$(curl -s -o /dev/null -w '%{http_code}' "$API/health" || true)
[ "$HC" = "200" ] || HC=$(curl -s -o /dev/null -w '%{http_code}' "$API/health/prices" || true)
if [ "$HC" = "200" ] || [ "$HC" = "401" ] || [ "$HC" = "404" ]; then
  ok "backend raggiungibile (http $HC)"
else
  warn "health backend ha risposto $HC ($BASE_URL) — verifica che lo stack test sia su"
fi
pause

# --- FASE 1: register/login ---------------------------------------------------
note "FASE 1 — Registrazione e login utente"
EMAIL="epick@test.local"
PW="Password123!"
register "$EMAIL" "$PW" "Epic K"
TOK=$(login "$EMAIL" "$PW")
if [ -n "$TOK" ]; then
  ok "login riuscito ($EMAIL)"
else
  die "login fallito — impossibile proseguire"
fi
pause

# --- FASE 2: asset (due ETF azionari) -----------------------------------------
note "FASE 2 — Crea asset SMEA.MI e SXR8.DE (idempotente al rerun)"
create_asset() { # $1 ticker  $2 name -> prints "id code"
  local res body code
  res=$(curl -s -w '\n%{http_code}' -X POST "$API/assets" \
    -H "Authorization: Bearer $TOK" \
    -H 'Content-Type: application/json' \
    -d "{\"ticker\":\"$1\",\"name\":\"$2\",\"type\":\"etf\",\"currency\":\"EUR\",\"asset_class\":\"equity\"}")
  body="${res%$'\n'*}"; code="${res##*$'\n'}"
  printf '%s %s' "$(jq -r '.id // empty' <<<"$body")" "$code"
}

read -r AI1 C1 <<<"$(create_asset "SMEA.MI" "iShares Core MSCI Europe UCITS ETF EUR (Acc)")"
read -r AI2 C2 <<<"$(create_asset "SXR8.DE" "iShares Core S&P 500 UCITS ETF EUR (Acc)")"
[ -n "$AI1" ] && ok "asset SMEA.MI pronto (id=$AI1, http $C1)" || bad "asset SMEA.MI non creato (http $C1)"
[ -n "$AI2" ] && ok "asset SXR8.DE pronto (id=$AI2, http $C2)" || bad "asset SXR8.DE non creato (http $C2)"
[ -n "$AI1" ] && [ -n "$AI2" ] || die "impossibile determinare gli id degli asset"
pause

# --- FASE 3: exposure sintetica via PUT (deterministica, nessun provider) ------
note "FASE 3 — Persisti exposure sintetica via PUT (drill deterministico)"
put_exposure() { # $1 asset_id  $2 label  $3 json
  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' -X PUT "$API/assets/$1/exposure" \
    -H "Authorization: Bearer $TOK" \
    -H 'Content-Type: application/json' \
    -d "$3")
  [ "$code" = "200" ] && ok "$2: exposure salvata (http 200)" || bad "$2: PUT exposure -> $code (atteso 200)"
}
# SMEA.MI: IT40/FR30/DE30, region Europe Developed 100, sectors 50/50.
put_exposure "$AI1" "SMEA.MI" '{
  "countries":[{"name":"IT","weight":40},{"name":"FR","weight":30},{"name":"DE","weight":30}],
  "regions":[{"name":"Europe Developed","weight":100}],
  "sectors":[{"name":"Financials","weight":50},{"name":"Industrials","weight":50}]
}'
# SXR8.DE: US100, region North America 100, sectors 60/40.
put_exposure "$AI2" "SXR8.DE" '{
  "countries":[{"name":"US","weight":100}],
  "regions":[{"name":"North America","weight":100}],
  "sectors":[{"name":"Information Technology","weight":60},{"name":"Health Care","weight":40}]
}'
pause

# --- FASE 4: seed prezzi (opzionale, non fatale) ------------------------------
SEEDED=0
if [ "$SEED" -eq 1 ]; then
  note "FASE 4 — Seed prezzi manuale (solo stack test, NON fatale)"
  if [ "$_PORT" = "8081" ]; then
    SEED_CMD=($COMPOSE -p peculium-test -f "$SCRIPT_DIR/../docker-compose.test.yml" exec -T postgres psql -U peculium -d peculium_test)
    if "${SEED_CMD[@]}" < "$SCRIPT_DIR/seed-prices.sql" >/dev/null 2>&1; then
      SEEDED=1
      ok "seed prezzi applicato (SMEA.MI @105, SXR8.DE @310)"
    else
      warn "seed prezzi fallito (container postgres test non attivo?) — i totali del drill resteranno 0"
    fi
  else
    warn "seed skipped (BASE_URL non sulla porta 8081)"
  fi
else
  note "FASE 4 — Seed prezzi saltato (--no-seed)"
fi

# Atteso con seed: SMEA 10*105 = 1050, SXR8 5*310 = 1550, totale 2600.
V1=1050; V2=1550; VTOT=2600
pause

# --- FASE 5: portafoglio + transazioni (buy x2, dividend x1) -------------------
note "FASE 5 — Portafoglio EUR + transazioni (buy SMEA, buy SXR8, dividend SMEA)"
EXISTING_PF=$(curl -s "$API/portfolios" -H "Authorization: Bearer $TOK")
PID=$(jq -r 'if type == "array" then map(select(.name == "EPIC K portfolio"))[0].id // empty else empty end' <<<"$EXISTING_PF")
if [ -n "$PID" ]; then
  ok "portafoglio esistente riusato ($PID)"
else
  PF=$(curl -s -X POST "$API/portfolios" \
    -H "Authorization: Bearer $TOK" \
    -H 'Content-Type: application/json' \
    -d '{"name":"EPIC K portfolio","description":"EPIC K smoke test","currency":"EUR"}')
  PID=$(jq -r '.id // empty' <<<"$PF")
  [ -n "$PID" ] && ok "portafoglio creato ($PID)" || die "creazione portafoglio fallita: $PF"
fi

TXS=$(curl -s "$API/portfolios/$PID/transactions" -H "Authorization: Bearer $TOK")
if [ "$(jq '.total' <<<"$TXS")" -gt 0 ]; then
  ok "transazioni gia presenti (rerun) — creazione saltata"
else
  add_tx() { # $1 asset_id  $2 type  $3 qty  $4 price  $5 date -> code
    curl -s -o /dev/null -w '%{http_code}' -X POST "$API/portfolios/$PID/transactions" \
      -H "Authorization: Bearer $TOK" \
      -H 'Content-Type: application/json' \
      -d "{\"asset_id\":\"$1\",\"type\":\"$2\",\"quantity\":$3,\"price\":$4,\"date\":\"$5\"}"
  }
  T1=$(add_tx "$AI1" buy 10 100.00 "2024-01-15T00:00:00Z")
  T2=$(add_tx "$AI2" buy 5 300.00 "2024-01-20T00:00:00Z")
  T3=$(add_tx "$AI1" dividend 1 50.00 "2024-02-10T00:00:00Z")
  [ "$T1" = "201" ] && ok "buy SMEA.MI -> 201" || bad "buy SMEA.MI -> $T1 (atteso 201)"
  [ "$T2" = "201" ] && ok "buy SXR8.DE -> 201" || bad "buy SXR8.DE -> $T2 (atteso 201)"
  [ "$T3" = "201" ] && ok "dividend SMEA.MI -> 201" || warn "dividend SMEA.MI -> $T3 (atteso 201) — i check su type=dividend verranno saltati"
fi
pause

# --- FASE 6: filtri transazioni (K.4c) ----------------------------------------
note "FASE 6 — Filtri elenco transazioni (type / asset_id / from / to)"
tx_total() { # $1 = raw query string (without ?) -> prints .total
  local q=""
  [ -n "$1" ] && q="?$1"
  curl -s "$API/portfolios/$PID/transactions$q" -H "Authorization: Bearer $TOK" | jq -r '.total // empty'
}

T_ALL=$(tx_total "")
[ "$T_ALL" = "3" ] && ok "senza filtri: total=3" || warn "senza filtri: total=$T_ALL (atteso 3; se >3 il rerun ha dati extra)"

T_BUY=$(tx_total "type=buy")
[ "$T_BUY" = "2" ] && ok "type=buy -> total=2" || bad "type=buy -> total=$T_BUY (atteso 2)"

T_ASSET=$(tx_total "asset_id=$AI1")
[ "$T_ASSET" = "2" ] && ok "asset_id=SMEA -> total=2 (buy+dividend)" || bad "asset_id=SMEA -> total=$T_ASSET (atteso 2)"

T_WIN=$(tx_total "from=2024-02-01&to=2024-02-28")
[ "$T_WIN" = "1" ] && ok "from/to febbraio -> total=1 (dividend)" || bad "from/to febbraio -> total=$T_WIN (atteso 1)"

T_COMB=$(tx_total "type=buy&from=2024-01-01&to=2024-01-31")
[ "$T_COMB" = "2" ] && ok "type=buy + finestra gennaio -> total=2" || bad "type=buy + finestra gennaio -> total=$T_COMB (atteso 2)"

# Il dividend e' il piu recente: la prima pagina ordinata deve averlo in testa.
T_FIRST=$(curl -s "$API/portfolios/$PID/transactions?type=dividend" -H "Authorization: Bearer $TOK" | jq -r '.transactions[0].type // empty')
[ "$T_FIRST" = "dividend" ] && ok "type=dividend: la riga restituita e' di tipo dividend" || warn "type=dividend: tipo riga = '$T_FIRST'"

# Input non validi -> 400.
C_BADTYPE=$(curl -s -o /dev/null -w '%{http_code}' "$API/portfolios/$PID/transactions?type=bogus" -H "Authorization: Bearer $TOK")
[ "$C_BADTYPE" = "400" ] && ok "type=bogus -> 400" || bad "type=bogus -> $C_BADTYPE (atteso 400)"

C_BADDATE=$(curl -s -o /dev/null -w '%{http_code}' "$API/portfolios/$PID/transactions?from=2024-13-40" -H "Authorization: Bearer $TOK")
[ "$C_BADDATE" = "400" ] && ok "from=2024-13-40 -> 400" || bad "from=2024-13-40 -> $C_BADDATE (atteso 400)"

C_BADASSET=$(curl -s -o /dev/null -w '%{http_code}' "$API/portfolios/$PID/transactions?asset_id=not-a-uuid" -H "Authorization: Bearer $TOK")
[ "$C_BADASSET" = "400" ] && ok "asset_id non-UUID -> 400" || bad "asset_id non-UUID -> $C_BADASSET (atteso 400)"
pause

# --- helper drill-down --------------------------------------------------------
# drill_assert: fetch a drill and check structure + total (+ optional expected).
# $1 base ("portfolio"|"dashboard") $2 dim $3 key $4 expected_total_or_empty
drill_assert() {
  local base="$1" dim="$2" key="$3" exp="$4" url body code
  local ekey; ekey="$(urlenc "$key")"
  if [ "$base" = "dashboard" ]; then
    url="$API/dashboard/allocation/drill?dim=$dim&key=$ekey"
  else
    url="$API/portfolios/$PID/allocation/drill?dim=$dim&key=$ekey"
  fi
  body=$(curl -s -w '\n%{http_code}' "$url" -H "Authorization: Bearer $TOK")
  code="${body##*$'\n'}"; body="${body%$'\n'*}"
  if [ "$code" != "200" ]; then
    bad "$base drill $dim=$key -> http $code"
    return
  fi
  # struttura
  if ! jq -e 'has("currency") and has("dim") and has("key") and has("total")
              and (.assets|type=="array")
              and (all(.assets[]?; has("asset_id") and has("ticker") and has("value")
                                     and has("weight") and has("contribution")))' \
        >/dev/null 2>&1 <<<"$body"; then
    bad "$base drill $dim=$key: struttura inattesa"
    return
  fi
  local got; got=$(jq -r '.total' <<<"$body")
  if [ -z "$exp" ]; then
    ok "$base drill $dim=$key: struttura ok (total=$got, assets=$(jq '.assets|length' <<<"$body"))"
    return
  fi
  if num_eq "$got" "$exp" 1; then
    ok "$base drill $dim=$key: total=$got (atteso ~$exp, assets=$(jq '.assets|length' <<<"$body"))"
  else
    bad "$base drill $dim=$key: total=$got (atteso ~$exp)"
  fi
  # contribution = value * weight / 100 per ogni asset
  local bad_row
  bad_row=$(jq -r '
    [ .assets[]? | select(
        ((.value|tonumber) * (.weight|tonumber) / 100 - (.contribution|tonumber)) | fabs > 0.01
      ) ] | length' <<<"$body" 2>/dev/null || echo 0)
  [ "$bad_row" = "0" ] && ok "$base drill $dim=$key: contribution = value*weight/100" \
                       || bad "$base drill $dim=$key: $bad_row righe con contribution incoerente"
}

# --- FASE 7: drill per classe (K.5) -------------------------------------------
note "FASE 7 — Drill-down per classe + riconciliazione con /allocation/class"
CLASS=$(curl -s "$API/portfolios/$PID/allocation/class" -H "Authorization: Bearer $TOK")
CV=$(bucket_value class "equity" <<<"$CLASS")
if [ "$SEEDED" -eq 1 ]; then
  num_eq "$CV" "$VTOT" 1 && ok "class equity value=$CV (atteso ~$VTOT)" || warn "class equity value=$CV (atteso ~$VTOT con seed)"
  drill_assert portfolio class "equity" "$CV"
  drill_assert dashboard class "equity" "$CV"
else
  warn "seed assente: salto i check numerici del drill per classe"
  drill_assert portfolio class "equity" ""
fi
pause

# --- FASE 8: drill per paese --------------------------------------------------
note "FASE 8 — Drill-down per paese (US da SXR8) + riconciliazione geography"
GEO=$(curl -s "$API/portfolios/$PID/allocation/geography" -H "Authorization: Bearer $TOK")
US_V=$(bucket_value country "US" <<<"$GEO")
if [ "$SEEDED" -eq 1 ]; then
  num_eq "$US_V" "$V2" 1 && ok "geography US value=$US_V (atteso ~$V2)" || warn "geography US value=$US_V (atteso ~$V2)"
  drill_assert portfolio country "US" "$US_V"
  # il drill US deve contenere un solo asset (SXR8) con contributo = V2
  DRILL_US=$(curl -s "$API/portfolios/$PID/allocation/drill?dim=country&key=US" -H "Authorization: Bearer $TOK")
  N_US=$(jq '.assets|length' <<<"$DRILL_US")
  [ "$N_US" = "1" ] && ok "drill country=US: un solo asset contribuente" || bad "drill country=US: $N_US asset (atteso 1)"
  C_US=$(jq -r '.assets[0].contribution' <<<"$DRILL_US")
  num_eq "$C_US" "$V2" 1 && ok "drill country=US: contribution=$C_US (atteso ~$V2)" || bad "drill country=US: contribution=$C_US (atteso ~$V2)"
  IT_V=$(bucket_value country "IT" <<<"$GEO")
  drill_assert portfolio country "IT" "$IT_V"
else
  warn "seed assente: salto i check numerici del drill per paese"
  drill_assert portfolio country "US" ""
fi
pause

# --- FASE 9: drill per regione ------------------------------------------------
note "FASE 9 — Drill-down per regione (North America da SXR8)"
NA_V=$(bucket_value region "North America" <<<"$GEO")
if [ "$SEEDED" -eq 1 ]; then
  drill_assert portfolio region "North America" "$NA_V"
  drill_assert dashboard region "North America" "$NA_V"
else
  drill_assert portfolio region "North America" ""
fi
pause

# --- FASE 10: drill per settore -----------------------------------------------
note "FASE 10 — Drill-down per settore (Information Technology da SXR8)"
SEC=$(curl -s "$API/portfolios/$PID/allocation/sector" -H "Authorization: Bearer $TOK")
IT_SEC_V=$(bucket_value sector "Information Technology" <<<"$SEC")
if [ "$SEEDED" -eq 1 ]; then
  drill_assert portfolio sector "Information Technology" "$IT_SEC_V"
else
  drill_assert portfolio sector "Information Technology" ""
fi
pause

# --- FASE 11: input non validi sul drill + ownership ---------------------------
note "FASE 11 — Drill-down: input non validi e ownership"
C_DIM=$(curl -s -o /dev/null -w '%{http_code}' "$API/portfolios/$PID/allocation/drill?dim=bogus&key=US" -H "Authorization: Bearer $TOK")
[ "$C_DIM" = "400" ] && ok "dim=bogus -> 400" || bad "dim=bogus -> $C_DIM (atteso 400)"

C_KEY=$(curl -s -o /dev/null -w '%{http_code}' "$API/portfolios/$PID/allocation/drill?dim=country&key=" -H "Authorization: Bearer $TOK")
[ "$C_KEY" = "400" ] && ok "key vuota -> 400" || bad "key vuota -> $C_KEY (atteso 400)"

C_NOPF=$(curl -s -o /dev/null -w '%{http_code}' "$API/portfolios/00000000-0000-0000-0000-000000000000/allocation/drill?dim=country&key=US" -H "Authorization: Bearer $TOK")
[ "$C_NOPF" = "404" ] && ok "portafoglio inesistente -> 404" || bad "portafoglio inesistente -> $C_NOPF (atteso 404)"

C_NOAUTH=$(curl -s -o /dev/null -w '%{http_code}' "$API/dashboard/allocation/drill?dim=country&key=US")
[ "$C_NOAUTH" = "401" ] && ok "drill dashboard senza token -> 401" || bad "drill dashboard senza token -> $C_NOAUTH (atteso 401)"
pause

# --- riepilogo ----------------------------------------------------------------
printf '\n\033[1mRisultato: %d PASS, %d FAIL (%d warning)\033[0m\n' "$PASS" "$FAIL" "$WARN"
exit "$([ "$FAIL" -eq 0 ] && echo 0 || echo 1)"
