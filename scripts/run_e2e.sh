#!/usr/bin/env bash
# Run E2E tests against TiDB (local or TiDB Cloud) and Upstash Redis.
# Set env vars before calling, or rely on defaults below.
#
# Required:
#   TEST_TIDB_DSN   full DSN, e.g. user:pass@tcp(host:4000)/chatsphere_test?parseTime=true&tls=tidb
#   OR individually:
#   TEST_TIDB_HOST  TEST_TIDB_PORT  TEST_TIDB_USER  TEST_TIDB_PASS  TEST_TIDB_NAME
#
#   TEST_REDIS_ADDR  e.g. star-crane-115594.upstash.io:6379
#   TEST_REDIS_PASS  Upstash password
#
# Example (TiDB Serverless):
#   TEST_TIDB_DSN="user:pass@tcp(gateway01.ap-northeast-1.prod.aws.tidbcloud.com:4000)/chatsphere_test?parseTime=true&tls=tidb" \
#   TEST_REDIS_ADDR="star-crane-115594.upstash.io:6379" \
#   TEST_REDIS_PASS="gQAAAAAAAcOK..." \
#   ./scripts/run_e2e.sh

set -euo pipefail

cd "$(dirname "$0")/.."

echo "=== ChatSphere E2E Tests ==="
echo "TiDB:  ${TEST_TIDB_DSN:-${TEST_TIDB_HOST:-127.0.0.1}:${TEST_TIDB_PORT:-4000}/${TEST_TIDB_NAME:-chatsphere_test}}"
echo "Redis: ${TEST_REDIS_ADDR:-(not set — tests will skip)}"
echo ""

go test -v -timeout 60s \
  ./internal/handler/... \
  ./internal/ws/... \
  2>&1 | tee /tmp/chatsphere-e2e.log

echo ""
echo "=== Done. Full log: /tmp/chatsphere-e2e.log ==="
