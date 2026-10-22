#!/usr/bin/env bash
# Smoke test for EPIC J.2/J.3 on the isolated test stack.
set -uo pipefail

BASE_URL="${1:-http://localhost:8081}"
API="$BASE_URL/api/v1"
EMAIL="epicj-$(date +%s)@test.local"
PW="password123"

fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }
check() { [[ "$1" == "$2" ]] || fail "$3 (got '$1', want '$2')"; }

curl -fsS -X POST "$API/auth/register" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PW\",\"name\":\"Epic J\"}" >/dev/null || fail "register"
TOKEN=$(curl -fsS -X POST "$API/auth/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PW\"}" | jq -r '.access_token // empty')
[[ -n "$TOKEN" ]] || fail "login"

create_asset() {
  curl -sS -w '\n%{http_code}' -X POST "$API/assets" \
    -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "$1"
}

TICKER="JBD$(date +%s)"
RESPONSE=$(create_asset "{\"ticker\":\"$TICKER\",\"name\":\"J test bond\",\"type\":\"bond\",\"currency\":\"EUR\",\"price_source\":\"none\",\"issuer_country\":\"IT\"}")
BODY=${RESPONSE%$'\n'*}
STATUS=${RESPONSE##*$'\n'}
check "$STATUS" 201 "create bond"
ID=$(jq -r '.id' <<<"$BODY")
check "$(jq -r '.asset_class' <<<"$BODY")" bond "bond default class"
check "$(jq -r '.issuer_country' <<<"$BODY")" IT "issuer country persisted"

EXPOSURE=$(curl -fsS "$API/assets/$ID/exposure" -H "Authorization: Bearer $TOKEN") || fail "read exposure"
check "$(jq -r '.countries[] | select(.name == "IT") | .weight' <<<"$EXPOSURE")" 100 "issuer country exposure seeded"
check "$(jq -r '.provenance.countries.source' <<<"$EXPOSURE")" issuer_country "issuer provenance"

PATCH=$(curl -sS -w '\n%{http_code}' -X PATCH "$API/assets/$ID" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"issuer":"Republic of Italy","issuer_country":"DE","maturity_date":"2032-05-15","attributes":{"coupon_rate":"3.5","rating":"AAA"}}')
BODY=${PATCH%$'\n'*}
STATUS=${PATCH##*$'\n'}
check "$STATUS" 200 "patch fixed-income metadata"
check "$(jq -r '.maturity_date | .[:10]' <<<"$BODY")" 2032-05-15 "maturity date persisted"
check "$(jq -r '.attributes.coupon_rate' <<<"$BODY")" 3.5 "JSONB attributes persisted"
EXPOSURE=$(curl -fsS "$API/assets/$ID/exposure" -H "Authorization: Bearer $TOKEN") || fail "read updated exposure"
check "$(jq -r '.countries[] | select(.weight == "100") | .name' <<<"$EXPOSURE")" DE "issuer country change updates derived exposure"

# Verify the repository query carries issuer_country through to actual portfolio
# allocation, while the sector endpoint still excludes fixed-income holdings.
TODAY=$(date -u +%F)
curl -fsS -X PUT "$API/assets/$ID/prices/manual" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d "{\"date\":\"$TODAY\",\"close\":100}" >/dev/null || fail "add manual bond price"
PF=$(curl -fsS -X POST "$API/portfolios" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"EPIC J geography","currency":"EUR"}')
PID=$(jq -r '.id // empty' <<<"$PF")
[[ -n "$PID" ]] || fail "create fixed-income test portfolio"
curl -fsS -X POST "$API/portfolios/$PID/transactions" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"asset_id\":\"$ID\",\"type\":\"buy\",\"quantity\":1,\"price\":100,\"date\":\"${TODAY}T00:00:00Z\"}" >/dev/null || fail "buy bond"
GEO=$(curl -fsS "$API/portfolios/$PID/allocation/geography" -H "Authorization: Bearer $TOKEN") || fail "read geography allocation"
check "$(jq -r '.countries[] | select(.country == "DE") | .value' <<<"$GEO")" 100 "bond value contributes to issuer-country geography"
check "$(jq -r '.covered_value' <<<"$GEO")" 100 "geography coverage includes fixed income"
SECTORS=$(curl -fsS "$API/portfolios/$PID/allocation/sector" -H "Authorization: Bearer $TOKEN") || fail "read sector allocation"
check "$(jq -r '.excluded_value' <<<"$SECTORS")" 100 "sector allocation excludes fixed income"

RESPONSE=$(create_asset "{\"ticker\":\"JCE$(date +%s)\",\"name\":\"J test certificate\",\"type\":\"certificate\",\"currency\":\"EUR\",\"price_source\":\"none\"}")
BODY=${RESPONSE%$'\n'*}
STATUS=${RESPONSE##*$'\n'}
check "$STATUS" 201 "create certificate"
check "$(jq -r '.type' <<<"$BODY")" certificate "certificate type persisted"
check "$(jq -r '.asset_class' <<<"$BODY")" other "certificate defaults to other class"

INVALID=$(curl -sS -o /dev/null -w '%{http_code}' -X PATCH "$API/assets/$ID" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"issuer_country":"not-a-country"}')
check "$INVALID" 400 "invalid issuer country rejected"

# A stock's geography is inferred 100% from its own country: the input is
# normalized to ISO alpha-2 on create, and with no stored country breakdown
# the position lands in the inferred country bucket — never in "Other".
STOCK=$(create_asset "{\"ticker\":\"JST$(date +%s)\",\"name\":\"J test stock\",\"type\":\"stock\",\"currency\":\"EUR\",\"price_source\":\"none\",\"country\":\"it\"}")
BODY=${STOCK%$'\n'*}
STATUS=${STOCK##*$'\n'}
check "$STATUS" 201 "create stock with country"
check "$(jq -r '.country' <<<"$BODY")" IT "stock country normalized to ISO alpha-2"
SID=$(jq -r '.id' <<<"$BODY")

curl -fsS -X PUT "$API/assets/$SID/prices/manual" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d "{\"date\":\"$TODAY\",\"close\":100}" >/dev/null || fail "add manual stock price"
SPF=$(curl -fsS -X POST "$API/portfolios" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"EPIC J stock country","currency":"EUR"}')
SPID=$(jq -r '.id // empty' <<<"$SPF")
[[ -n "$SPID" ]] || fail "create stock-country portfolio"
curl -fsS -X POST "$API/portfolios/$SPID/transactions" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"asset_id\":\"$SID\",\"type\":\"buy\",\"quantity\":1,\"price\":100,\"date\":\"${TODAY}T00:00:00Z\"}" >/dev/null || fail "buy stock"
SGEO=$(curl -fsS "$API/portfolios/$SPID/allocation/geography" -H "Authorization: Bearer $TOKEN") || fail "read stock geography allocation"
check "$(jq -r '.countries[] | select(.country == "IT") | .value | tonumber' <<<"$SGEO")" 100 "stock geography inferred from asset country"
check "$(jq -r '[.countries[] | select(.country == "Other")] | length' <<<"$SGEO")" 0 "inferred country never falls into the Other bucket"

# ── Option A1: asset-level quote divisor ──────────────────────────────────
# A bond quoted per 100 nominal: quantity stays the nominal, the quote stays
# the quote, and every cash amount is quantity × price / quote_divisor.
QD_RESPONSE=$(create_asset "{\"ticker\":\"JQD$(date +%s)\",\"name\":\"J divisor bond\",\"type\":\"bond\",\"currency\":\"EUR\",\"price_source\":\"none\",\"issuer_country\":\"IT\",\"quote_divisor\":100}")
QD_BODY=${QD_RESPONSE%$'\n'*}
QD_STATUS=${QD_RESPONSE##*$'\n'}
check "$QD_STATUS" 201 "create bond with quote_divisor"
check "$(jq -r '.quote_divisor' <<<"$QD_BODY")" 100 "quote divisor echoed as a decimal string"
QD_ID=$(jq -r '.id' <<<"$QD_BODY")
for BAD in '{"quote_divisor":0}' '{"quote_divisor":-5}' '{"quote_divisor":"abc"}'; do
  INVALID=$(curl -sS -o /dev/null -w '%{http_code}' -X PATCH "$API/assets/$QD_ID" \
    -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "$BAD")
  check "$INVALID" 400 "invalid quote_divisor $BAD rejected"
done
GET_BACK=$(curl -fsS "$API/assets/$QD_ID" -H "Authorization: Bearer $TOKEN")
check "$(jq -r '.quote_divisor' <<<"$GET_BACK")" 100 "quote divisor survives the GET round-trip"

curl -fsS -X PUT "$API/assets/$QD_ID/prices/manual" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d "{\"date\":\"$TODAY\",\"close\":100.25}" >/dev/null || fail "add divisor bond price"
QDPF=$(curl -fsS -X POST "$API/portfolios" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"EPIC J quote divisor","currency":"EUR"}')
QDPID=$(jq -r '.id // empty' <<<"$QDPF")
[[ -n "$QDPID" ]] || fail "create quote-divisor portfolio"
curl -fsS -X POST "$API/portfolios/$QDPID/transactions" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"asset_id\":\"$QD_ID\",\"type\":\"buy\",\"quantity\":10000,\"price\":100.25,\"date\":\"${TODAY}T00:00:00Z\"}" >/dev/null || fail "buy 10000 nominal at 100.25"

# Nominal × quote / 100 = 10000 × 100.25 / 100 = 10025 everywhere money moves.
SUMMARY=$(curl -fsS "$API/portfolios/$QDPID/summary" -H "Authorization: Bearer $TOKEN") || fail "read divisor summary"
check "$(jq -r '.total_value | tonumber' <<<"$SUMMARY")" 10025 "portfolio value is nominal × quote / divisor"
GEO=$(curl -fsS "$API/portfolios/$QDPID/allocation/geography" -H "Authorization: Bearer $TOKEN") || fail "read divisor geography"
check "$(jq -r '.countries[] | select(.country == "IT") | .value | tonumber' <<<"$GEO")" 10025 "geography values are divided"
PERF=$(curl -fsS "$API/portfolios/$QDPID/performance" -H "Authorization: Bearer $TOKEN") || fail "read divisor performance"
check "$(jq -r '.[-1].value | tonumber' <<<"$PERF")" 10025 "materialized series carries the divided value"
EXPORT=$(curl -fsS "$API/portfolios/$QDPID/export" -H "Authorization: Bearer $TOKEN") || fail "read divisor export"
check "$(jq -r '.assets[] | select(.ticker != null) | .quote_divisor' <<<"$EXPORT")" 100 "quote divisor travels in the portfolio export"

# A plain asset (default divisor 1) still values at quantity × price: the
# first bond in this script bought 1 unit at 100 must still sum to 100.
BASE_SUMMARY=$(curl -fsS "$API/portfolios/$PID/summary" -H "Authorization: Bearer $TOKEN") || fail "read default-divisor summary"
check "$(jq -r '.total_value | tonumber' <<<"$BASE_SUMMARY")" 100 "divisor 1 keeps quantity × price"

printf 'EPIC J.2/J.3 smoke test: PASS\n'
