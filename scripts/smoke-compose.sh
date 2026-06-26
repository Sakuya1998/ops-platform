#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="deploy/docker-compose.yml"
APISIX_BASE_URL="http://localhost:9080"
USERNAME="admin"
PASSWORD="admin@2026"

usage() {
  cat <<'EOF'
Usage: scripts/smoke-compose.sh [options]

Options:
  --compose-file FILE       Docker Compose file, default: deploy/docker-compose.yml
  --apisix-base-url URL     APISIX base URL, default: http://localhost:9080
  --username USER           Login username, default: admin
  --password PASSWORD       Login password, default: admin@2026
  -h, --help                Show help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose-file)
      COMPOSE_FILE="$2"
      shift 2
      ;;
    --apisix-base-url)
      APISIX_BASE_URL="$2"
      shift 2
      ;;
    --username)
      USERNAME="$2"
      shift 2
      ;;
    --password)
      PASSWORD="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Required command '$1' was not found in PATH" >&2
    exit 1
  fi
}

wait_http_ok() {
  local url="$1"
  local timeout_seconds="${2:-180}"
  local deadline=$((SECONDS + timeout_seconds))
  while (( SECONDS < deadline )); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 3
  done
  echo "Timed out waiting for $url" >&2
  return 1
}

extract_token() {
  python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["access_token"])'
}

require_command docker
require_command curl
require_command python3

cd "$REPO_ROOT"

echo "=== Building and starting Compose stack ==="
docker compose -f "$COMPOSE_FILE" up -d --build

echo "=== Waiting for APISIX health endpoint ==="
wait_http_ok "$APISIX_BASE_URL/health"

echo "=== Seeding default organization, admin user, roles, and permissions ==="
"$REPO_ROOT/scripts/seed.sh" --use-docker-compose --compose-file "$COMPOSE_FILE"

echo "=== Login through APISIX ==="
login_payload="$(python3 -c 'import json,sys; print(json.dumps({"username": sys.argv[1], "password": sys.argv[2], "org_code": "default", "provider": "local"}))' "$USERNAME" "$PASSWORD")"
login_response="$(curl -fsS -X POST "$APISIX_BASE_URL/api/v1/auth/login" -H 'Content-Type: application/json' -d "$login_payload")"
token="$(printf '%s' "$login_response" | extract_token)"
if [[ -z "$token" ]]; then
  echo "Login response did not contain data.access_token: $login_response" >&2
  exit 1
fi

echo "=== Verifying core APIs through APISIX ==="
curl -fsS "$APISIX_BASE_URL/api/v1/auth/me" -H "Authorization: Bearer $token" >/dev/null
curl -fsS "$APISIX_BASE_URL/api/v1/users" -H "Authorization: Bearer $token" >/dev/null
curl -fsS "$APISIX_BASE_URL/api/v1/roles" -H "Authorization: Bearer $token" >/dev/null
curl -fsS "$APISIX_BASE_URL/api/v1/audit-logs" -H "Authorization: Bearer $token" >/dev/null

echo "=== Smoke test passed ==="
