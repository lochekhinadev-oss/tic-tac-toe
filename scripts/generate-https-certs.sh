#!/usr/bin/env sh
set -eu

OUT_DIR="${1:-./certs}"
CERT_FILE="$OUT_DIR/tic-tac-toe.crt"
KEY_FILE="$OUT_DIR/tic-tac-toe.key"
CA_KEY_FILE="$OUT_DIR/tic-tac-toe-root-ca.key"
CA_CERT_FILE="$OUT_DIR/tic-tac-toe-root-ca.crt"
CA_DER_FILE="$OUT_DIR/tic-tac-toe-root-ca.der"
CSR_FILE="$OUT_DIR/tic-tac-toe.csr"
SAN_CONF="$OUT_DIR/openssl-san.cnf"
CA_CONF="$OUT_DIR/openssl-ca.cnf"

mkdir -p "$OUT_DIR"

if [ "$(uname -s)" = "Darwin" ] && command -v security >/dev/null 2>&1; then
  TRUST_CA=1
else
  TRUST_CA=0
fi

cat >"$SAN_CONF" <<'EOF'
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = req_ext

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

cat >"$CA_CONF" <<'EOF'
[req]
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = v3_ca

[dn]
C = RU
ST = Moscow
L = Moscow
O = TicTacToe
OU = Development
CN = TicTacToe Local CA

[v3_ca]
basicConstraints = critical,CA:TRUE
keyUsage = critical,keyCertSign,cRLSign
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer
EOF

openssl genrsa -out "$CA_KEY_FILE" 2048
openssl req -x509 -new -nodes \
  -key "$CA_KEY_FILE" \
  -sha256 \
  -days 3650 \
  -out "$CA_CERT_FILE" \
  -config "$CA_CONF"
openssl x509 -in "$CA_CERT_FILE" -outform der -out "$CA_DER_FILE"

if [ "$TRUST_CA" -eq 1 ]; then
  if ! security add-trusted-cert -r trustRoot -p ssl -k "$HOME/Library/Keychains/login.keychain-db" "$CA_DER_FILE" >/dev/null 2>&1; then
    printf 'User keychain trust import failed, retrying system keychain.\n' >&2
    security add-trusted-cert -d -r trustRoot -p ssl -k /Library/Keychains/System.keychain "$CA_DER_FILE"
  fi
else
  printf 'Warning: certificate authority was created but could not be trusted automatically on this platform.\n' >&2
  printf 'The HTTPS stack will still work, but browsers may warn about the certificate.\n' >&2
fi

openssl req -new \
  -keyout "$KEY_FILE" \
  -out "$CSR_FILE" \
  -nodes \
  -subj "/C=RU/ST=Moscow/L=Moscow/O=TicTacToe/OU=Development/CN=localhost" \
  -config "$SAN_CONF"

openssl x509 -req \
  -in "$CSR_FILE" \
  -CA "$CA_CERT_FILE" \
  -CAkey "$CA_KEY_FILE" \
  -CAcreateserial \
  -CAserial "$OUT_DIR/tic-tac-toe-root-ca.srl" \
  -out "$CERT_FILE" \
  -days 825 \
  -sha256 \
  -extfile "$SAN_CONF" \
  -extensions req_ext

rm -f "$CSR_FILE" "$SAN_CONF" "$CA_CONF" "$CA_CERT_FILE.srl" "$OUT_DIR/tic-tac-toe-root-ca.srl" "$CA_KEY_FILE" "$CA_CERT_FILE" "$CA_DER_FILE"
chmod 600 "$KEY_FILE"
chmod 644 "$CERT_FILE"

printf 'Generated HTTPS certificate: %s\n' "$CERT_FILE"
printf 'Generated HTTPS private key: %s\n' "$KEY_FILE"
