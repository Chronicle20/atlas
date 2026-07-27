#!/usr/bin/env bats
# test/reconcile_minio_test.bats

setup() {
  SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/../scripts" && pwd)"
  export TMP="$BATS_TEST_TMPDIR"
  cat >"$TMP/kubectl" <<'EOF'
#!/usr/bin/env bash
echo "namespace/atlas-main"
echo "namespace/atlas-pr-42"
EOF
  chmod +x "$TMP/kubectl"
  export KUBECTL="$TMP/kubectl"
}

@test "unions tenant ids across namespaces and posts keep-list" {
  cat >"$TMP/curl" <<EOF
#!/usr/bin/env bash
args="\$*"
case "\$args" in
  *atlas-main*tenants*)  echo '{"data":[{"id":"aaaa"}]}'; exit 0 ;;
  *atlas-pr-42*tenants*) echo '{"data":[{"id":"bbbb"}]}'; exit 0 ;;
  *minio/reconcile*)     echo "\$args" >>"$TMP/posted"; echo '{"totalBytes":0}'; exit 0 ;;
esac
exit 0
EOF
  chmod +x "$TMP/curl"; export CURL="$TMP/curl"
  run "$SCRIPT_DIR/reconcile-minio.sh"
  [ "$status" -eq 0 ]
  grep -q '"aaaa"' "$TMP/posted"
  grep -q '"bbbb"' "$TMP/posted"
  # ParseTenant requires the tenant headers on the POST (else atlas-data 400s)
  grep -q 'TENANT_ID' "$TMP/posted"
  grep -q 'MAJOR_VERSION' "$TMP/posted"
}

@test "fail-closed: unreachable namespace aborts without POST" {
  cat >"$TMP/curl" <<EOF
#!/usr/bin/env bash
args="\$*"
case "\$args" in
  *atlas-main*tenants*)  echo '{"data":[{"id":"aaaa"}]}'; exit 0 ;;
  *atlas-pr-42*tenants*) exit 7 ;;
  *minio/reconcile*)     echo "posted" >>"$TMP/posted"; exit 0 ;;
esac
exit 0
EOF
  chmod +x "$TMP/curl"; export CURL="$TMP/curl"
  run "$SCRIPT_DIR/reconcile-minio.sh"
  [ "$status" -ne 0 ]
  [ ! -f "$TMP/posted" ]
}

@test "refuses empty union" {
  cat >"$TMP/curl" <<EOF
#!/usr/bin/env bash
args="\$*"
case "\$args" in
  *tenants*)         echo '{"data":[]}'; exit 0 ;;
  *minio/reconcile*) echo "posted" >>"$TMP/posted"; exit 0 ;;
esac
exit 0
EOF
  chmod +x "$TMP/curl"; export CURL="$TMP/curl"
  run "$SCRIPT_DIR/reconcile-minio.sh"
  [ "$status" -ne 0 ]
  [ ! -f "$TMP/posted" ]
}
