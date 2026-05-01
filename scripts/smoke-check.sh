#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8090}"
USERNAME="${UPAY_ADMIN_USERNAME:-admin}"
PASSWORD="${UPAY_ADMIN_PASSWORD:-admin}"
COOKIE_FILE="$(mktemp)"

cleanup() {
  rm -f "$COOKIE_FILE"
}
trap cleanup EXIT

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

pass() {
  echo "OK: $1"
}

expect_json_code_zero() {
  local label="$1"
  local body="$2"

  if ! echo "$body" | grep -q '"code":0'; then
    echo "$body" >&2
    fail "$label did not return code=0"
  fi
}

health_body="$(curl -fsS "$BASE_URL/healthz")" || fail "healthz request failed"
expect_json_code_zero "healthz" "$health_body"
if ! echo "$health_body" | grep -q '"status":"ok"'; then
  echo "$health_body" >&2
  fail "healthz status is not ok"
fi
pass "healthz"

auth_config_body="$(curl -fsS "$BASE_URL/login/auth-config")" || fail "auth config request failed"
expect_json_code_zero "auth config" "$auth_config_body"
pass "login auth config"

login_body="$(
  curl -fsS \
    -c "$COOKIE_FILE" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" \
    "$BASE_URL/login"
)" || fail "password login request failed"
expect_json_code_zero "password login" "$login_body"
pass "password login"

stats_body="$(curl -fsS -b "$COOKIE_FILE" "$BASE_URL/admin/api/stats")" || fail "admin stats request failed"
expect_json_code_zero "admin stats" "$stats_body"
pass "admin stats"

operations_body="$(curl -fsS -b "$COOKIE_FILE" "$BASE_URL/admin/api/operations/summary")" || fail "operations summary request failed"
expect_json_code_zero "operations summary" "$operations_body"
if ! echo "$operations_body" | grep -q '"orders"'; then
  echo "$operations_body" >&2
  fail "operations summary missing orders"
fi
pass "operations summary"

passkeys_body="$(curl -fsS -b "$COOKIE_FILE" "$BASE_URL/admin/api/passkeys")" || fail "passkey settings request failed"
expect_json_code_zero "passkey settings" "$passkeys_body"
pass "passkey settings"

echo "Smoke check completed against $BASE_URL"
