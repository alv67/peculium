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

printf 'EPIC J.2/J.3 smoke test: PASS\n'
