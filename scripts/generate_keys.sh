#!/bin/bash

set -e

KEYS_DIR="${1:-keys}"
KEY_SIZE="${2:-2048}"

mkdir -p "$KEYS_DIR"

echo "Generating RSA key pair in '$KEYS_DIR'..."

openssl genrsa -out "$KEYS_DIR/private.pem" $KEY_SIZE
openssl rsa -in "$KEYS_DIR/private.pem" -pubout -out "$KEYS_DIR/public.pem"

chmod 600 "$KEYS_DIR/private.pem"
chmod 644 "$KEYS_DIR/public.pem"

echo "Keys generated successfully:"
echo "  Private key: $KEYS_DIR/private.pem"
echo "  Public key:  $KEYS_DIR/public.pem"
echo ""
echo "Important: Add '$KEYS_DIR/private.pem' to your .gitignore!"
