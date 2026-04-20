#!/usr/bin/env bash
#
# Integration test: generate + sign a LoTE using SoftHSM2
# Mimics the trust-lists GitHub Actions workflow exactly.
#
# Usage: ./test/integration/softhsm_lote_test.sh
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

echo "=== Integration test: SoftHSM2 LoTE signing ==="
echo "Work dir: $WORK_DIR"
echo "Repo root: $REPO_ROOT"

# ── 1. Build tsl-tool ──────────────────────────────────────────────
echo ""
echo "--- Building tsl-tool ---"
cd "$REPO_ROOT"
go build -o "$WORK_DIR/tsl-tool" ./cmd/tsl-tool
export PATH="$WORK_DIR:$PATH"

# ── 2. Set up ephemeral SoftHSM2 token ─────────────────────────────
echo ""
echo "--- Setting up SoftHSM2 ---"
TOKEN_DIR="$WORK_DIR/softhsm"
mkdir -p "$TOKEN_DIR"

cat > "$WORK_DIR/softhsm2.conf" <<CONF
directories.tokendir = $TOKEN_DIR
objectstore.backend = file
log.level = INFO
slots.removable = true
CONF
export SOFTHSM2_CONF="$WORK_DIR/softhsm2.conf"

softhsm2-util --init-token --free \
  --label "trust-lists" \
  --so-pin 5678 \
  --pin 1234

# Find PKCS#11 module
for p in /usr/lib/softhsm/libsofthsm2.so /usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so /usr/lib64/softhsm/libsofthsm2.so; do
  if [ -f "$p" ]; then PKCS11_MODULE="$p"; break; fi
done
echo "Module: $PKCS11_MODULE"

# Generate EC P-256 key pair
pkcs11-tool --module "$PKCS11_MODULE" \
  --token-label "trust-lists" \
  --login --pin 1234 \
  --keypairgen --key-type ec:prime256v1 \
  --id 01 --label "signing-key"

# Generate self-signed certificate
openssl req -new -x509 -nodes \
  -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
  -keyout "$WORK_DIR/temp-key.pem" \
  -out "$WORK_DIR/cert.pem" \
  -days 365 \
  -subj "/CN=Integration Test Signer"

# Import certificate into token
pkcs11-tool --module "$PKCS11_MODULE" \
  --token-label "trust-lists" \
  --login --pin 1234 \
  --write-object "$WORK_DIR/cert.pem" --type cert \
  --id 01 --label "signing-cert"

echo ""
echo "--- Token objects ---"
pkcs11-tool --module "$PKCS11_MODULE" --token-label "trust-lists" --list-objects

# ── 3. Set up test LoTE source ─────────────────────────────────────
echo ""
echo "--- Setting up test data ---"
LISTS_DIR="$WORK_DIR/lists/test-demo"
mkdir -p "$LISTS_DIR/entities/test-issuer"

cat > "$LISTS_DIR/scheme.yaml" <<'YAML'
operatorNames:
  - language: en
    value: "Test Operator"
schemeName:
  - language: en
    value: "Test Trust List"
schemeType: "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric"
territory: "SE"
sequenceNumber: 1
YAML

cat > "$LISTS_DIR/entities/test-issuer/entity.yaml" <<'YAML'
names:
  - language: en
    value: "Test Issuer"
entityId: "https://test.example.com"
status: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted"
services:
  - serviceNames:
      - language: en
        value: "Test Service"
    serviceType: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC"
    status: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted"
YAML

# ── 4. Build and run the pipeline ──────────────────────────────────
echo ""
echo "--- Running pipeline ---"
OUTPUT_DIR="$WORK_DIR/dist"
PKCS11_URI="pkcs11:module=${PKCS11_MODULE};pin=1234;token=trust-lists"

PIPELINE="$WORK_DIR/pipeline.yaml"
{
  echo "- generate-lote:"
  echo "    - $LISTS_DIR/"
  echo "- increment-lote-sequence: []"
  echo "- publish-lote:"
  echo "    - $OUTPUT_DIR"
  echo "    - $PKCS11_URI"
  echo "    - signing-key"
  echo "    - signing-cert"
  echo "    - 01"
} > "$PIPELINE"

echo "Pipeline content:"
cat "$PIPELINE"
echo ""

tsl-tool "$PIPELINE"

# ── 5. Verify output ───────────────────────────────────────────────
echo ""
echo "--- Verifying output ---"
ls -la "$OUTPUT_DIR/"

if ls "$OUTPUT_DIR"/*.jws 1>/dev/null 2>&1; then
  echo ""
  echo "=== SUCCESS: Signed LoTE files produced ==="
  for f in "$OUTPUT_DIR"/*.jws; do
    echo "  $f ($(wc -c < "$f") bytes)"
  done
else
  echo ""
  echo "=== FAILURE: No .jws files produced ==="
  exit 1
fi

if ls "$OUTPUT_DIR"/*.json 1>/dev/null 2>&1; then
  echo "  Unsigned JSON also present"
fi

echo ""
echo "=== Integration test PASSED ==="
