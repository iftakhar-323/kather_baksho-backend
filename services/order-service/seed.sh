#!/usr/bin/env bash
# Full demo dataset for KatherBox. Idempotent — safe to re-run.
# Usage:  ./seed.sh  [product_count]  [user_count]
set -e
cd "$(dirname "$0")"

PRODUCTS="${1:-120}"
USERS="${2:-60}"

echo "==> makeadmin"
go run ./cmd/makeadmin/

echo "==> seedproducts ($PRODUCTS)"
go run ./cmd/seedproducts/ "$PRODUCTS"

echo "==> seedusers ($USERS)"
go run ./cmd/seedusers/ "$USERS"

echo "==> seeddummy (orders, reviews, returns, subscriptions, addresses, …)"
go run ./cmd/seeddummy/

echo "==> seedorders (extra orders for customer@test.com)"
go run ./cmd/seedorders/ || true

echo
echo "Done. Admin: admin@katherbox.com / Admin@12345"
echo "      Seeded users: <name>@katherbox.test / Test@12345"
