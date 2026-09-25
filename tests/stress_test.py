#!/usr/bin/env python3
"""
Kather Baksho - Automated Concurrency & Stress Testing Benchmark
Simulates high-velocity Flash Sale concurrency scenarios.
Tests ACID atomicity, race conditions, zero-overselling guarantees, and latency distribution.
"""

import argparse
import concurrent.futures
import math
import sys
import time
import requests

def run_flash_sale_benchmark(base_url="http://localhost:8081/api", concurrency=50, initial_stock=5):
    print("\n" + "=" * 70)
    print(f"🔥 KATHER BAKSHO FLASH SALE CONCURRENCY BENCHMARK")
    print("=" * 70)
    print(f"  API Base URL:   {base_url}")
    print(f"  Competitors:    {concurrency} concurrent threads")
    print(f"  Limited Stock:  {initial_stock} units available")
    print("-" * 70)

    session = requests.Session()

    # 1. Admin login to create product
    print("[1/5] Authenticating admin and provisioning flash sale product...")
    r_login = session.post(f"{base_url}/auth/login", json={
        "email": "admin@kather_baksho.com",
        "password": "Admin@12345"
    })
    if r_login.status_code != 200:
        print(f"Admin login failed: {r_login.text}")
        return False

    admin_token = r_login.json().get("token")
    admin_headers = {
        "Authorization": f"Bearer {admin_token}",
        "X-Benchmark": "true"
    }

    timestamp = int(time.time() * 1000)
    product_slug = f"flash-bonsai-{timestamp}"
    r_prod = session.post(f"{base_url}/products/", json={
        "name": f"Flash Sale Rare Bonsai #{timestamp}",
        "slug": product_slug,
        "description": "High-demand limited edition bonsai for concurrency benchmark",
        "price": 1500.00,
        "stock": initial_stock,
        "category": "plant"
    }, headers=admin_headers)

    if r_prod.status_code not in (200, 201):
        print(f"Failed to create benchmark product: {r_prod.text}")
        return False

    product_data = r_prod.json().get("product", r_prod.json())
    product_id = product_data.get("id") or product_data.get("ID")
    print(f"  ✓ Created Product ID {product_id} with initial stock = {initial_stock}")

    # 2. Provision distinct user tokens for each competitor
    print(f"[2/5] Registering {concurrency} distinct customer accounts...")
    user_tokens = []
    
    def register_user(idx):
        u_email = f"flash_racer_{timestamp}_{idx}@bench.local"
        r = requests.post(f"{base_url}/auth/register", json={
            "name": f"Racer #{idx}",
            "email": u_email,
            "password": "Password123!"
        }, headers={"X-Benchmark": "true"}, timeout=10)
        if r.status_code == 201:
            return r.json().get("token")
        return None

    with concurrent.futures.ThreadPoolExecutor(max_workers=min(concurrency, 20)) as executor:
        tokens = list(executor.map(register_user, range(concurrency)))
        user_tokens = [t for t in tokens if t]

    if len(user_tokens) < concurrency:
        print(f"Warning: Only {len(user_tokens)}/{concurrency} accounts registered. Continuing with available.")
        concurrency = len(user_tokens)

    print(f"  ✓ {concurrency} competitor tokens ready.")

    # 3. Synchronized Flash Sale Attack
    print(f"[3/5] Synchronizing and firing {concurrency} concurrent buy requests...")
    barrier = concurrent.futures.Barrier = None
    results = []

    def attempt_purchase(token_and_id):
        tok, racer_id = token_and_id
        headers = {
            "Authorization": f"Bearer {tok}",
            "X-Benchmark": "true"
        }
        start_t = time.perf_counter()
        try:
            r = requests.post(f"{base_url}/cart/add", json={
                "product_id": product_id,
                "quantity": 1
            }, headers=headers, timeout=10)
            latency = (time.perf_counter() - start_t) * 1000.0
            return {
                "racer_id": racer_id,
                "status_code": r.status_code,
                "latency_ms": latency,
                "response": r.json() if r.headers.get("content-type", "").startswith("application/json") else r.text
            }
        except Exception as ex:
            latency = (time.perf_counter() - start_t) * 1000.0
            return {
                "racer_id": racer_id,
                "status_code": 0,
                "latency_ms": latency,
                "error": str(ex)
            }

    race_start = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as pool:
        args = [(user_tokens[i], i + 1) for i in range(concurrency)]
        results = list(pool.map(attempt_purchase, args))
    total_duration = time.perf_counter() - race_start

    # 4. Analyze Results
    print(f"[4/5] Analyzing ACID atomicity and transaction results...")
    successful_buys = [r for r in results if r["status_code"] in (200, 201)]
    out_of_stock = [r for r in results if r["status_code"] == 400]
    other_errors = [r for r in results if r["status_code"] not in (200, 400)]

    # Fetch final product state from DB
    r_check = session.get(f"{base_url}/products/{product_id}")
    final_prod = r_check.json()
    final_stock = final_prod.get("stock", -1)

    latencies = sorted([r["latency_ms"] for r in results])
    def percentile(p):
        idx = int(math.ceil((p / 100.0) * len(latencies))) - 1
        return latencies[max(0, min(idx, len(latencies) - 1))]

    rps = len(results) / total_duration if total_duration > 0 else 0

    print("-" * 70)
    print("📊 BENCHMARK METRICS SUMMARY")
    print("-" * 70)
    print(f"  Total Requests:         {len(results)}")
    print(f"  Successful Checkouts:   {len(successful_buys)} (Expected: {initial_stock})")
    print(f"  Rejected (Out-of-Stock):{len(out_of_stock)} (Expected: {concurrency - initial_stock})")
    print(f"  Other Errors:           {len(other_errors)}")
    print(f"  Final Product Stock:    {final_stock} units (Expected: 0)")
    print(f"  Oversold Stock Units:   {max(0, len(successful_buys) - initial_stock)}")
    print(f"  Total Time:             {total_duration * 1000:.2f} ms")
    print(f"  Throughput:             {rps:.1f} req/sec")
    print(f"  Latency Min:            {min(latencies):.2f} ms")
    print(f"  Latency p50 (Median):   {percentile(50):.2f} ms")
    print(f"  Latency p95:            {percentile(95):.2f} ms")
    print(f"  Latency p99:            {percentile(99):.2f} ms")
    print(f"  Latency Max:            {max(latencies):.2f} ms")
    print("-" * 70)

    # 5. Clean up product
    print("[5/5] Cleaning up test product...")
    session.delete(f"{base_url}/products/{product_id}", headers=admin_headers)

    # Assertions
    success = True
    if len(successful_buys) != initial_stock:
        print(f"❌ FAIL: Expected {initial_stock} successful checkouts, got {len(successful_buys)}")
        success = False
    elif final_stock != 0:
        print(f"❌ FAIL: Expected final stock = 0, got {final_stock}")
        success = False
    elif len(other_errors) > 0:
        print(f"❌ FAIL: Encountered {len(other_errors)} unexpected errors")
        success = False
    else:
        print("✅ PASS: 100% ACID Atomicity Confirmed. ZERO Overselling!")

    print("=" * 70 + "\n")
    return success

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Kather Baksho Concurrency Benchmark")
    parser.add_argument("--concurrency", type=int, default=30, help="Number of concurrent competitors")
    parser.add_argument("--stock", type=int, default=5, help="Available stock units")
    parser.add_argument("--url", type=str, default="http://localhost:8081/api", help="API URL")
    args = parser.parse_args()

    ok = run_flash_sale_benchmark(base_url=args.url, concurrency=args.concurrency, initial_stock=args.stock)
    sys.exit(0 if ok else 1)
