#!/usr/bin/env bash
# Build, vet and test the data-dynamodb plugin's tree as it composes over api-go, then run the
# data-store@1 conformance suite against DynamoDB Local.
#
#   verify-data-dynamodb.sh [--no-dynamodb]
#
# The real composer lives in albitor-ltd/albitor and is not importable here, so this lays the two
# plugins' templates/api trees over each other (the paths do not overlap) and renders api-go's
# generated router with every slot empty, as verify-api-go.sh does. scripts/compose-check.sh and
# tests/compose are the authority on what composes.
#
# DynamoDB Local: when DYNAMODB_LOCAL_ENDPOINT is set that endpoint is used; otherwise the script
# starts amazon/dynamodb-local on a loopback port and removes it afterwards. --no-dynamodb skips
# it (the memory adapter's conformance still runs). The managed tree carries no DynamoDB Local
# test, so an app's own CI needs no Docker: conformance/dynamodb_local_test.go is copied in here.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
apigo="$root/plugins/api-go"
plugin="$root/plugins/data-dynamodb"
image="amazon/dynamodb-local:3.0.0"
use_dynamodb=1
[[ "${1:-}" == "--no-dynamodb" ]] && use_dynamodb=0

work="$(mktemp -d)"
container=""
cleanup() {
  if [[ -n "$container" ]]; then docker rm -f "$container" >/dev/null 2>&1; fi
  rm -rf "$work"
}
trap cleanup EXIT

cp -R "$apigo/templates/api/." "$work/"
cp -R "$plugin/templates/api/." "$work/"
sed -E 's/\{\{albitor\.slot\.[a-z.-]+\}\}//g' \
  "$apigo/slots/api/internal/shell/router.go.tmpl" >"$work/internal/shell/router.go"

cd "$work"
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  echo "error: not gofmt-clean:" >&2
  echo "$unformatted" >&2
  exit 1
fi

# The module stays stdlib only: the plugin must not need a dependency api-go's go.mod lacks.
if grep -q '^require' go.mod; then
  echo "error: go.mod gained a dependency; the store must stay stdlib-only" >&2
  exit 1
fi

go build ./...
go vet ./...
go test ./... -count=1

if [[ $use_dynamodb -eq 0 ]]; then
  echo "skipped DynamoDB Local conformance (--no-dynamodb)"
  exit 0
fi

endpoint="${DYNAMODB_LOCAL_ENDPOINT:-}"
if [[ -z "$endpoint" ]]; then
  command -v docker >/dev/null || { echo "error: docker is required for DynamoDB Local" >&2; exit 2; }
  container="$(docker run -d --rm -p 127.0.0.1::8000 "$image" -jar DynamoDBLocal.jar -inMemory -sharedDb)"
  port="$(docker port "$container" 8000/tcp | head -n1 | sed 's/.*://')"
  endpoint="http://127.0.0.1:$port"
  for _ in $(seq 1 60); do
    if curl -s -o /dev/null "$endpoint"; then break; fi
    sleep 0.5
  done
fi

cp "$plugin/conformance/dynamodb_local_test.go" internal/store/dynamodb_local_test.go
DYNAMODB_LOCAL_ENDPOINT="$endpoint" go test ./internal/store/ -run 'TestDynamoDBLocal' -count=1 -v
