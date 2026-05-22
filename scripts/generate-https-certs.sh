#!/usr/bin/env sh
set -eu

OUT_DIR="${1:-./certs}"
CERT_FILE="$OUT_DIR/tic-tac-toe.crt"
KEY_FILE="$OUT_DIR/tic-tac-toe.key"
TMP_CONF="$OUT_DIR/openssl-san.cnf"

mkdir -p "$OUT_DIR"

if command -v mkcert >/dev/null 2>&1; then
  mkcert -cert-file "$CERT_FILE" -key-file "$KEY_FILE" localhost 127.0.0.1 ::1
  chmod 600 "$KEY_FILE"
  chmod 644 "$CERT_FILE"
  printf 'Generated HTTPS certificate via mkcert: %s\n' "$CERT_FILE"
  printf 'Generated HTTPS private key via mkcert: %s\n' "$KEY_FILE"
  exit 0
fi

cat >"$TMP_CONF" <<'EOF'
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = req_ext

[dn]
C = RU
ST = Moscow
L = Moscow
O = TicTacToe
OU = Development
CN = localhost

[req_ext]
subjectAltName = @alt_names
basicConstraints = critical,CA:FALSE
keyUsage = critical,digitalSignature,keyEncipherment
extendedKeyUsage = serverAuth

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

openssl req -x509 -nodes -newkey rsa:2048 \
  -keyout "$KEY_FILE" \
  -out "$CERT_FILE" \
  -days 3650 \
  -config "$TMP_CONF" \
  -extensions req_ext

rm -f "$TMP_CONF"
chmod 600 "$KEY_FILE"
chmod 644 "$CERT_FILE"

printf 'Generated HTTPS certificate: %s\n' "$CERT_FILE"
printf 'Generated HTTPS private key: %s\n' "$KEY_FILE"
