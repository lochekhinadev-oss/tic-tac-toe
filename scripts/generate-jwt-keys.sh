#!/usr/bin/env sh
set -eu

OUT_DIR="${1:-./secrets/jwt}"
PRIVATE_KEY="$OUT_DIR/private.pem"
PUBLIC_KEY="$OUT_DIR/public.pem"

mkdir -p "$OUT_DIR"

openssl genrsa -out "$PRIVATE_KEY" 2048
openssl rsa -in "$PRIVATE_KEY" -pubout -out "$PUBLIC_KEY"

chmod 600 "$PRIVATE_KEY"
chmod 644 "$PUBLIC_KEY"

printf 'Generated JWT private key: %s\n' "$PRIVATE_KEY"
printf 'Generated JWT public key: %s\n' "$PUBLIC_KEY"
