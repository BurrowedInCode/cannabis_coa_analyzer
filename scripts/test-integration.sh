#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Explicit environment variables take precedence over the local configuration.
if [[ -z "${TEST_DATABASE_URL:-}" && -f "$repo_root/.env.test" ]]; then
  set -a
  source "$repo_root/.env.test"
  set +a
fi

if [[ -z "${TEST_DATABASE_URL:-}" ]]; then
  echo "Set TEST_DATABASE_URL or copy .env.test.example to .env.test." >&2
  exit 1
fi
export TEST_DATABASE_URL

cd "$repo_root/server"
exec go test -tags=integration ./... -count=1 "$@"
