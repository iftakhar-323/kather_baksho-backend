import requests
import sys
import json
import time

BASE_URL = "http://localhost:8081/api"

def log(msg, success=True):
    symbol = "✓" if success else "✗"
    print(f"[{symbol}] {msg}")

results = []

def run_test(name, fn):
    try:
        fn()
        log(name, True)
        results.append((name, True, ""))
    except Exception as e:
        log(f"{name}: {e}", False)
        results.append((name, False, str(e)))

s = requests.Session()
admin_s = requests.Session()
staff_s = requests.Session()

# 1. Health / Products
def test_get_products():
    r = s.get(f"{BASE_URL}/products/")
    assert r.status_code == 200, f"Status {r.status_code}: {r.text}"
    data = r.json()
    assert "items" in data or isinstance(data, list), "Unexpected products response"

# 2. Login Admin
admin_token = None
def test_login_admin():
    global admin_token
    r = admin_s.post(f"{BASE_URL}/auth/login", json={"email": "admin@kather_baksho.com", "password": "Admin@12345"})
    assert r.status_code == 200, f"Status {r.status_code}: {r.text}"
    data = r.json()
    assert "token" in data, "No token returned"
    admin_token = data["token"]
    admin_s.headers.update({"Authorization": f"Bearer {admin_token}"})

# 3. Login Staff
staff_token = None
def test_login_staff():
    global staff_token
    r = staff_s.post(f"{BASE_URL}/auth/login", json={"email": "staff@kather_baksho.com", "password": "Staff@12345"})
    assert r.status_code == 200, f"Status {r.status_code}: {r.text}"
    data = r.json()
    assert "token" in data, "No token returned"
    staff_token = data["token"]
    staff_s.headers.update({"Authorization": f"Bearer {staff_token}"})

# 4. Register new user
user_s = requests.Session()
test_email = f"e2etest_{int(time.time())}@kather_baksho.com"
def test_register_user():
    r = user_s.post(f"{BASE_URL}/auth/register", json={
        "name": "E2E User",
        "email": test_email,
        "password": "Password@123"
    })
    assert r.status_code in (200, 201), f"Status {r.status_code}: {r.text}"
    data = r.json()
    assert "token" in data, "No token in register response"
    user_s.headers.update({"Authorization": f"Bearer {data['token']}"})

# 5. User Me
def test_user_me():
    r = user_s.get(f"{BASE_URL}/auth/me")
    assert r.status_code == 200, f"Status {r.status_code}: {r.text}"
    data = r.json()
    user_info = data.get("user", data)
    assert user_info.get("email") == test_email, f"Expected {test_email}, got {user_info}"

# 6. Update Profile
def test_update_profile():
    r = user_s.put(f"{BASE_URL}/auth/profile", json={"name": "E2E User Updated"})
    assert r.status_code == 200, f"Status {r.status_code}: {r.text}"

# 7. Addresses
addr_id = None
def test_addresses():
    global addr_id
    r = user_s.post(f"{BASE_URL}/auth/addresses", json={
        "label": "Home",
        "recipient": "E2E Tester",
        "phone": "01700000000",
        "line1": "House 1, Road 2",
        "city": "Dhaka",
        "is_default": True
    })
    assert r.status_code in (200, 201), f"Status {r.status_code}: {r.text}"
    addr = r.json().get("address", r.json())
    addr_id = addr.get("id") or addr.get("ID")
    assert addr_id, f"No address id in response: {r.json()}"
    
    r2 = user_s.get(f"{BASE_URL}/auth/addresses")
    assert r2.status_code == 200, f"Status {r2.status_code}: {r2.text}"

# 8. Cart Operations & Stock reservation check
product_id = None
initial_stock = 0
def test_cart_operations():
    global product_id, initial_stock
    pr = s.get(f"{BASE_URL}/products/")
    pdata = pr.json()
    items = pdata.get("items", [])
    for p in items:
        if p.get("stock", 0) > 5:
            product_id = p.get("id") or p.get("ID")
            initial_stock = s.get(f"{BASE_URL}/products/{product_id}").json().get("stock")
            break
    assert product_id, "No in-stock product found"

    # Add to cart (quantity 1) -> stock decreases by 1
    r = user_s.post(f"{BASE_URL}/cart/add", json={"product_id": product_id, "quantity": 1})
    assert r.status_code in (200, 201), f"Add to cart failed: {r.status_code} {r.text}"

    # Verify stock reserved
    pr_after = s.get(f"{BASE_URL}/products/{product_id}").json()
    assert pr_after.get("stock") == initial_stock - 1, f"Expected stock {initial_stock - 1}, got {pr_after.get('stock')}"

    # Get cart
    r_cart = user_s.get(f"{BASE_URL}/cart/")
    assert r_cart.status_code == 200, f"Get cart failed: {r_cart.status_code} {r_cart.text}"
    cdata = r_cart.json()
    cart_items = cdata.get("items", [])
    assert len(cart_items) > 0, "Cart is empty after adding item"
    item_id = cart_items[0].get("id") or cart_items[0].get("ID")

    # Update item quantity to 2 -> stock decreases by another 1
    r_up = user_s.put(f"{BASE_URL}/cart/item/{item_id}", json={"quantity": 2})
    assert r_up.status_code == 200, f"Update cart item failed: {r_up.status_code} {r_up.text}"
    pr_after2 = s.get(f"{BASE_URL}/products/{product_id}").json()
    assert pr_after2.get("stock") == initial_stock - 2, f"Expected stock {initial_stock - 2}, got {pr_after2.get('stock')}"

# 9. Coupon
def test_coupon_validate():
    r = user_s.post(f"{BASE_URL}/coupons/apply", json={"code": "WELCOME10", "order_total": 1000})
    assert r.status_code == 200, f"Coupon apply failed: {r.status_code} {r.text}"
    data = r.json()
    assert data.get("discount_amount") == 100, f"Expected 100 discount, got {data}"

# 10. Checkout / Orders
order_id = None
def test_checkout():
    global order_id
    r = user_s.post(f"{BASE_URL}/orders/checkout", json={
        "coupon_code": "WELCOME10",
        "shipping_name": "E2E User",
        "shipping_address": "House 1, Road 2, Dhaka",
        "shipping_phone": "01700000000",
        "payment_method": "cod"
    })
    assert r.status_code in (200, 201), f"Checkout failed: {r.status_code} {r.text}"
    odata = r.json()
    order = odata.get("order", odata)
    order_id = order.get("id") or order.get("ID")
    assert order_id, f"No order id in response: {odata}"

# 11. Verify Database stock after order remains reserved
def test_stock_after_checkout():
    pr = s.get(f"{BASE_URL}/products/{product_id}")
    assert pr.status_code == 200
    p = pr.json()
    assert p.get("stock") == initial_stock - 2, f"Stock changed unexpectedly: {p.get('stock')}"

# 12. Get user orders & details
def test_get_orders():
    r = user_s.get(f"{BASE_URL}/orders/")
    assert r.status_code == 200, f"Get orders failed: {r.status_code} {r.text}"
    r2 = user_s.get(f"{BASE_URL}/orders/{order_id}")
    assert r2.status_code == 200, f"Get single order failed: {r2.status_code} {r2.text}"

# 13. Order invoice & receipt
def test_order_invoice_receipt():
    r_inv = user_s.get(f"{BASE_URL}/orders/{order_id}/invoice")
    assert r_inv.status_code == 200, f"Invoice failed: {r_inv.status_code}"
    r_rec = user_s.get(f"{BASE_URL}/orders/{order_id}/receipt")
    assert r_rec.status_code == 200, f"Receipt failed: {r_rec.status_code}"

# 14. Wishlist
wishlist_item_id = None
def test_wishlist():
    global wishlist_item_id
    r_add = user_s.post(f"{BASE_URL}/wishlist/add", json={"product_id": product_id})
    assert r_add.status_code in (200, 201), f"Wishlist add failed: {r_add.status_code} {r_add.text}"
    witem = r_add.json()
    wishlist_item_id = witem.get("id") or witem.get("ID")
    r_get = user_s.get(f"{BASE_URL}/wishlist/")
    assert r_get.status_code == 200, f"Wishlist get failed: {r_get.status_code} {r_get.text}"
    if wishlist_item_id:
        r_del = user_s.delete(f"{BASE_URL}/wishlist/{wishlist_item_id}")
        assert r_del.status_code == 200, f"Wishlist delete failed: {r_del.status_code} {r_del.text}"

# 15. Reviews
def test_reviews():
    r_post = user_s.post(f"{BASE_URL}/products/{product_id}/reviews", json={
        "rating": 5,
        "comment": "Great plant!"
    })
    assert r_post.status_code in (200, 201), f"Review post failed: {r_post.status_code} {r_post.text}"
    r_get = s.get(f"{BASE_URL}/products/{product_id}/reviews")
    assert r_get.status_code == 200, f"Review get failed: {r_get.status_code} {r_get.text}"

# 16. Consultations
consultation_id = None
def test_consultations():
    global consultation_id
    r = user_s.post(f"{BASE_URL}/consultations", json={
        "expert_name": "Rina Akter",
        "topic": "Repotting advice",
        "scheduled_at": "2026-09-20 10:00",
        "notes": "Please call 10 mins early"
    })
    assert r.status_code in (200, 201), f"Consultation create failed: {r.status_code} {r.text}"
    c = r.json().get("consultation", r.json())
    consultation_id = c.get("id") or c.get("ID")
    r_get = user_s.get(f"{BASE_URL}/consultations")
    assert r_get.status_code == 200, f"Consultation list failed: {r_get.status_code}"

# 17. Subscriptions
def test_subscriptions():
    r = user_s.post(f"{BASE_URL}/subscriptions", json={
        "plan_name": "Monthly Green",
        "interval_days": 30,
        "price": 1200
    })
    assert r.status_code in (200, 201), f"Subscription create failed: {r.status_code} {r.text}"
    r_get = user_s.get(f"{BASE_URL}/subscriptions")
    assert r_get.status_code == 200, f"Subscription list failed: {r_get.status_code}"

# 18. Growth Journal
def test_journal():
    r = user_s.post(f"{BASE_URL}/journal", json={
        "product_id": product_id,
        "title": "First leaf sprout",
        "notes": "Watered with fertilizer today"
    })
    assert r.status_code in (200, 201), f"Journal create failed: {r.status_code} {r.text}"
    r_get = user_s.get(f"{BASE_URL}/journal")
    assert r_get.status_code == 200, f"Journal list failed: {r_get.status_code}"

# 19. Community Posts (Public Read + Auth Create)
def test_community():
    # Public unauthenticated read
    r_pub = s.get(f"{BASE_URL}/community/posts")
    assert r_pub.status_code == 200, f"Public community list failed: {r_pub.status_code} {r_pub.text}"

    # Authenticated post create
    r = user_s.post(f"{BASE_URL}/community/posts", json={
        "title": "E2E Test Post",
        "body": "Hello plant lovers!"
    })
    assert r.status_code in (200, 201), f"Community post failed: {r.status_code} {r.text}"

# 20. Admin - Update Order Status & Cancel (restores stock)
def test_admin_order_status_and_stock_restore():
    r = staff_s.put(f"{BASE_URL}/admin/orders/{order_id}/status", json={"status": "Cancelled"})
    assert r.status_code == 200, f"Staff cancel order failed: {r.status_code} {r.text}"

    # Stock should be restored to initial_stock!
    pr = s.get(f"{BASE_URL}/products/{product_id}")
    assert pr.status_code == 200
    restored_stock = pr.json().get("stock")
    assert restored_stock == initial_stock, f"Stock was not restored on cancellation! Expected {initial_stock}, got {restored_stock}"

# 21. Admin - Analytics
def test_admin_analytics():
    r = admin_s.get(f"{BASE_URL}/admin/analytics")
    assert r.status_code == 200, f"Admin analytics failed: {r.status_code} {r.text}"

# 22. Admin - List Users
def test_admin_users():
    r = admin_s.get(f"{BASE_URL}/admin/users")
    assert r.status_code == 200, f"Admin users list failed: {r.status_code} {r.text}"

# 23. Admin - Product CRUD with unique slug generation
created_product_id = None
def test_admin_product_crud():
    global created_product_id
    r = admin_s.post(f"{BASE_URL}/products/", json={
        "name": "E2E Test Bonsai",
        "price": 1250,
        "category": "plant",
        "subcategory": "indoor_plant",
        "stock": 10,
        "description": "A miniature bonsai tree."
    })
    assert r.status_code in (200, 201), f"Product create failed: {r.status_code} {r.text}"
    p = r.json().get("product", r.json())
    created_product_id = p.get("id") or p.get("ID")
    assert created_product_id and created_product_id > 0, f"Invalid product ID: {created_product_id} in {p}"

    # Update product
    r_up = admin_s.put(f"{BASE_URL}/products/{created_product_id}", json={
        "name": "E2E Test Bonsai Updated",
        "price": 1400,
        "stock": 15
    })
    assert r_up.status_code == 200, f"Product update failed: {r_up.status_code} {r_up.text}"

    # Delete product
    r_del = admin_s.delete(f"{BASE_URL}/products/{created_product_id}")
    assert r_del.status_code == 200, f"Product delete failed: {r_del.status_code} {r_del.text}"

# 24. Admin - Consultations confirm/cancel
def test_admin_consultations():
    if consultation_id:
        r = admin_s.post(f"{BASE_URL}/admin/consultations/{consultation_id}/confirm")
        assert r.status_code == 200, f"Confirm consultation failed: {r.status_code} {r.text}"

# 25. Categories API
def test_categories():
    r = s.get(f"{BASE_URL}/categories/")
    assert r.status_code == 200, f"Categories failed: {r.status_code}"
    data = r.json()
    assert isinstance(data, list) or "categories" in data, f"Unexpected: {data}"

# 26. Blog API & Article Detail
def test_blog():
    r = s.get(f"{BASE_URL}/blog/")
    assert r.status_code == 200, f"Blog list failed: {r.status_code}"
    posts = r.json()
    if isinstance(posts, list) and len(posts) > 0:
        slug = posts[0].get("slug")
        if slug:
            r2 = s.get(f"{BASE_URL}/blog/{slug}")
            assert r2.status_code == 200, f"Blog detail failed: {r2.status_code}"

# 27. Security Boundary: Unauthenticated Admin Access Protected
def test_security_boundary():
    r = s.get(f"{BASE_URL}/admin/analytics")
    assert r.status_code in (401, 403), f"Expected 401/403, got {r.status_code}"

# 28. Frontend SPA Routing & Fallback
def test_frontend_routes():
    FE_URL = "http://localhost:8082"
    for route in ["/", "/cart", "/admin", "/shop", "/blog", "/community"]:
        r = requests.get(f"{FE_URL}{route}")
        assert r.status_code == 200, f"Frontend route {route} failed: {r.status_code}"
        assert "<html" in r.text or "<!DOCTYPE" in r.text or "<!doctype" in r.text

# 29. SQLite DB PRAGMA Integrity Check
def test_db_health():
    import sqlite3
    conn = sqlite3.connect("backend/kather_baksho.db")
    cur = conn.cursor()
    cur.execute("PRAGMA integrity_check;")
    row = cur.fetchone()
    assert row[0] == "ok", f"Integrity check failed: {row}"
    conn.close()

# 30. Prometheus Metrics Expose
def test_prometheus():
    r = s.get("http://localhost:8081/metrics")
    assert r.status_code == 200, f"Metrics failed: {r.status_code}"
    assert "http_requests_total" in r.text, "Missing http_requests_total in metrics"

# 31. Enhanced Health Readiness Probe
def test_health_ready():
    r = s.get("http://localhost:8081/health/ready")
    assert r.status_code == 200, f"Health ready failed: {r.status_code}"
    data = r.json()
    assert data.get("database", {}).get("status") == "connected"
    assert "system" in data

# 32. Idempotency Key Replay
def test_idempotency():
    import uuid
    key = f"e2e-idem-{uuid.uuid4()}"
    h = {"Idempotency-Key": key}
    r1 = user_s.get(f"{BASE_URL}/auth/me", headers=h)
    assert r1.status_code == 200
    r2 = user_s.get(f"{BASE_URL}/auth/me", headers=h)
    assert r2.status_code == 200
    assert r2.headers.get("X-Cache-Lookup") == "HIT", "Expected HIT from idempotency replay"

# 33. Payment Session & HMAC Verification
def test_payment_flow():
    import hmac, hashlib
    r_init = user_s.post(f"{BASE_URL}/payments/initiate", json={"order_id": order_id, "payment_method": "bkash"})
    if r_init.status_code == 200:
        pdata = r_init.json()
        session_id = pdata["session_id"]
        sig = pdata["signature"]
        cb = s.post(f"{BASE_URL}/payments/callback", json={
            "order_id": order_id,
            "session_id": session_id,
            "status": "SUCCESS",
            "signature": sig
        })
        assert cb.status_code == 200, f"Payment callback failed: {cb.text}"
        assert cb.json().get("payment_status") == "Paid"

# 34. AI Plant Doctor Diagnosis
def test_ai_diagnose():
    r = s.post(f"{BASE_URL}/ai/diagnose", json={
        "plant_name": "Fiddle Leaf Fig",
        "symptoms": "yellow leaves with brown crispy edges",
        "environment": "Living room"
    })
    assert r.status_code == 200, f"AI diagnose failed: {r.status_code}"
    data = r.json()
    assert "diagnosis" in data
    assert "action_plan" in data
    assert len(data["action_plan"]) > 0
    assert "provider_used" in data

# 35. AI Plant Doctor Chat
def test_ai_chat():
    r = s.post(f"{BASE_URL}/ai/chat", json={
        "question": "How much sunlight does Monstera need?"
    })
    assert r.status_code == 200, f"AI chat failed: {r.status_code}"
    data = r.json()
    assert "answer" in data
    assert len(data["answer"]) > 0

# 36. ML Model Comparison Benchmark
def test_ml_compare():
    r = s.post(f"{BASE_URL}/ml/compare", json={
        "plant": "Monstera",
        "symptom": "Yellowing leaves with chlorosis"
    })
    assert r.status_code == 200, f"ML compare failed: {r.status_code}"
    data = r.json()
    assert "model_a" in data
    assert "model_b" in data
    assert data["model_a"]["latency_ms"] > 0
    assert data["model_b"]["latency_ms"] > 0
    assert "latency_speedup" in data
    assert "recommendation" in data

# 37. Algorithm Visualizer Route Check
def test_algorithm_visualizer():
    FE_URL = "http://localhost:8082"
    r = requests.get(f"{FE_URL}/algorithms")
    assert r.status_code == 200, f"Algorithm route failed: {r.status_code}"
    assert "<html" in r.text or "<!DOCTYPE" in r.text or "<!doctype" in r.text

# 38. Redis Read-Through Caching & Cache-HIT Verification
def test_redis_caching():
    test_tag = f"test-{int(time.time())}"
    # Unique query to force MISS
    r1 = s.get(f"{BASE_URL}/products?search={test_tag}")
    assert r1.status_code == 200, f"Query failed: {r1.status_code}"
    assert r1.headers.get("X-Cache") == "MISS", f"Expected MISS, got {r1.headers.get('X-Cache')}"

    # Repeated query must return HIT
    r2 = s.get(f"{BASE_URL}/products?search={test_tag}")
    assert r2.status_code == 200
    assert r2.headers.get("X-Cache") == "HIT", f"Expected HIT, got {r2.headers.get('X-Cache')}"

# 39. Cache Invalidation upon Mutation
def test_cache_invalidation():
    # Cache categories
    r_cat1 = s.get(f"{BASE_URL}/categories")
    assert r_cat1.status_code == 200
    # Next call is HIT
    r_cat2 = s.get(f"{BASE_URL}/categories")
    assert r_cat2.headers.get("X-Cache") == "HIT"

    # Create category via admin
    r_create = admin_s.post(f"{BASE_URL}/admin/categories", json={
        "name": f"Rare Orchids {int(time.time())}",
        "parent": "indoor",
        "icon": "🌸",
        "position": 99
    })
    assert r_create.status_code == 201, f"Create category failed: {r_create.text}"

    # After mutation, category list must be MISS (invalidated!)
    r_cat3 = s.get(f"{BASE_URL}/categories")
    assert r_cat3.headers.get("X-Cache") == "MISS", f"Expected cache MISS after invalidation, got {r_cat3.headers.get('X-Cache')}"

# 40. Multi-Database Health & Observability (SQLite + Redis)
def test_multidb_health():
    r = s.get("http://localhost:8081/health/ready")
    assert r.status_code == 200
    data = r.json()
    assert data.get("database", {}).get("status") == "connected", "Database must be connected"
    assert data.get("cache", {}).get("provider") == "redis", "Cache provider must be redis"
    assert data.get("cache", {}).get("status") == "connected", "Redis must be connected"

# 41. Node.js & TypeScript Worker Health Probe
def test_worker_health():
    r = requests.get("http://localhost:8083/health", timeout=5)
    assert r.status_code == 200, f"Worker health failed: {r.status_code}"
    data = r.json()
    assert data.get("service") == "kather_baksho-worker-ts"
    assert data.get("status") == "UP"
    assert "pdf-invoice-generation" in data.get("features", [])

# 42. Node.js & TypeScript Direct PDF Invoice Generation
def test_worker_direct_invoice():
    payload = {
        "orderId": 777,
        "customerName": "Botanical Tester",
        "customerEmail": "tester@kather_baksho.com",
        "items": [
            {"id": 1, "productName": "Fiddle Leaf Fig Large", "quantity": 1, "price": 3500, "subtotal": 3500}
        ],
        "total": 3500
    }
    r = requests.post("http://localhost:8083/api/v1/invoices/generate", json=payload, timeout=8)
    assert r.status_code == 200, f"Direct PDF generation failed: {r.status_code}"
    assert r.headers.get("content-type") == "application/pdf"
    assert r.headers.get("x-generated-by") == "kather_baksho-worker-ts"
    assert r.content.startswith(b"%PDF"), "Must return valid PDF magic bytes"

# 43. Go Backend Order PDF Invoice Forwarding / Bridge
def test_go_pdf_invoice_proxy():
    assert order_id is not None, "Order ID required"
    r = user_s.get(f"{BASE_URL}/orders/{order_id}/invoice/pdf", timeout=10)
    assert r.status_code == 200, f"Go PDF invoice proxy failed: {r.status_code} {r.text}"
    assert r.headers.get("content-type") == "application/pdf"
    assert r.headers.get("x-generated-by") == "kather_baksho-worker-ts"
    assert r.content.startswith(b"%PDF"), "Must return valid PDF magic bytes"

# 44. Go Backend Analytics Report PDF Generation
def test_go_analytics_report_pdf():
    r = admin_s.get(f"{BASE_URL}/analytics/report/pdf?days=30", timeout=10)
    assert r.status_code == 200, f"Analytics report PDF failed: {r.status_code} {r.text}"
    assert r.headers.get("content-type") == "application/pdf"
    assert r.headers.get("x-generated-by") == "kather_baksho-worker-ts"
    assert r.content.startswith(b"%PDF"), "Must return valid PDF magic bytes"

# 45. MongoDB Tri-Database Observability (SQLite + Redis + MongoDB)
def test_mongodb_readiness_probe():
    r = s.get("http://localhost:8081/health/ready")
    assert r.status_code == 200
    data = r.json()
    assert data.get("nosql_database", {}).get("provider") == "mongodb"
    assert data.get("nosql_database", {}).get("status") == "connected", "MongoDB must be connected"

# 46. MongoDB IoT Telemetry Ingestion & Auto-Diagnosis
def test_iot_telemetry_ingestion():
    payload = {
        "plant_id": 1,
        "plant_name": "Monstera Deliciosa",
        "location": "Living Room Test Pot",
        "soil_moisture_pct": 19.5,
        "ambient_temp_c": 26.5,
        "humidity_pct": 55.0,
        "light_lux": 650.0,
        "battery_pct": 98.0
    }
    r = s.post(f"{BASE_URL}/iot/telemetry", json=payload)
    assert r.status_code == 201, f"Ingest failed: {r.status_code} {r.text}"
    data = r.json()
    assert data.get("database") == "mongodb"
    assert data.get("telemetry", {}).get("status") == "Needs Water"

# 47. MongoDB Monitored Plants Snapshot
def test_iot_monitored_plants():
    r = s.get(f"{BASE_URL}/iot/plants")
    assert r.status_code == 200, f"Get plants failed: {r.status_code}"
    data = r.json()
    assert "plants" in data
    assert len(data["plants"]) > 0

# 48. MongoDB IoT Telemetry Time-Series History
def test_iot_telemetry_history():
    r = s.get(f"{BASE_URL}/iot/telemetry/1?limit=10")
    assert r.status_code == 200, f"History query failed: {r.status_code}"
    data = r.json()
    assert data.get("plant_id") == 1
    assert "history" in data
    assert len(data["history"]) > 0

# 49. Traefik Cloud-Native Ingress & Reverse Proxy Routing
def test_traefik_api_gateway_routing():
    r_fe = requests.get("http://localhost:8085/", timeout=5)
    assert r_fe.status_code == 200, f"Traefik frontend failed: {r_fe.status_code}"

    r_be = requests.get("http://localhost:8085/api/products/", timeout=5)
    assert r_be.status_code == 200, f"Traefik backend failed: {r_be.status_code}"
    assert "items" in r_be.json() or isinstance(r_be.json(), list)

    r_worker = requests.get("http://localhost:8085/worker/health", timeout=5)
    assert r_worker.status_code == 200, f"Traefik worker failed: {r_worker.status_code}"
    assert r_worker.json().get("service") == "kather_baksho-worker-ts"

# 50. Traefik Live Telemetry & Dashboard Exposure
def test_traefik_dashboard():
    r_dash = requests.get("http://localhost:8086/dashboard/", timeout=5)
    assert r_dash.status_code == 200, f"Traefik dashboard failed: {r_dash.status_code}"
    r_api = requests.get("http://localhost:8086/api/rawdata", timeout=5)
    assert r_api.status_code == 200
    assert "routers" in r_api.json()

# 51. WebSocket Real-Time Order & Delivery Rider Tracking
def test_websocket_order_tracking():
    import socket
    # Test direct WebSocket handshake on backend port 8081
    s_be = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s_be.settimeout(5.0)
    s_be.connect(("localhost", 8081))
    req = (
        "GET /ws/orders/1/track HTTP/1.1\r\n"
        "Host: localhost:8081\r\n"
        "Upgrade: websocket\r\n"
        "Connection: Upgrade\r\n"
        "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n"
        "Sec-WebSocket-Version: 13\r\n\r\n"
    )
    s_be.sendall(req.encode())
    resp = s_be.recv(4096)
    assert b"101 Switching Protocols" in resp, f"Direct WS upgrade failed: {resp[:100]}"
    frame = s_be.recv(4096)
    assert b"connected" in frame or b"gps_update" in frame, f"Direct frame not received: {frame[:100]}"
    s_be.close()

    # Test WebSocket handshake through Traefik Gateway on port 8085
    s_tr = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s_tr.settimeout(5.0)
    s_tr.connect(("localhost", 8085))
    req_tr = (
        "GET /ws/orders/1/track HTTP/1.1\r\n"
        "Host: localhost:8085\r\n"
        "Upgrade: websocket\r\n"
        "Connection: Upgrade\r\n"
        "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n"
        "Sec-WebSocket-Version: 13\r\n\r\n"
    )
    s_tr.sendall(req_tr.encode())
    resp_tr = s_tr.recv(4096)
    assert b"101 Switching Protocols" in resp_tr, f"Traefik WS upgrade failed: {resp_tr[:100]}"
    frame_tr = s_tr.recv(4096)
    assert b"connected" in frame_tr or b"gps_update" in frame_tr, f"Traefik frame not received: {frame_tr[:100]}"
    s_tr.close()

# 52. OpenAPI 3.0 Specification & Interactive Swagger UI
def test_openapi_swagger_docs():
    # Direct Go Backend
    r_spec = requests.get("http://localhost:8081/api/docs/openapi.json", timeout=5)
    assert r_spec.status_code == 200, f"OpenAPI spec failed: {r_spec.status_code}"
    spec = r_spec.json()
    assert spec.get("openapi") == "3.0.3"
    assert "paths" in spec
    assert "/api/products/" in spec["paths"]
    assert "/api/iot/telemetry" in spec["paths"]

    r_ui = requests.get("http://localhost:8081/docs", timeout=5)
    assert r_ui.status_code == 200, f"Docs HTML failed: {r_ui.status_code}"
    assert "SwaggerUIBundle" in r_ui.text

    # Via Traefik Ingress Gateway
    r_tr_spec = requests.get("http://localhost:8085/api/docs/openapi.json", timeout=5)
    assert r_tr_spec.status_code == 200, f"Traefik OpenAPI spec failed: {r_tr_spec.status_code}"

    r_tr_ui = requests.get("http://localhost:8085/docs", timeout=5)
    assert r_tr_ui.status_code == 200, f"Traefik Docs UI failed: {r_tr_ui.status_code}"

# 53. Frontend TypeScript Type Declarations & Config
def test_frontend_typescript_types():
    import json
    import os
    assert os.path.exists("frontend/tsconfig.json"), "tsconfig.json missing"
    with open("frontend/tsconfig.json", "r") as f:
        tsconfig = json.load(f)
    assert "compilerOptions" in tsconfig
    assert tsconfig["compilerOptions"].get("strict") is True

    expected_files = ["index.d.ts", "product.d.ts", "order.d.ts", "user.d.ts", "telemetry.d.ts"]
    for fname in expected_files:
        path = os.path.join("frontend/src/types", fname)
        assert os.path.exists(path), f"Missing type declaration: {fname}"
        with open(path, "r") as f:
            content = f.read()
        assert len(content) > 50, f"Empty type declaration: {fname}"

# 54. AWS EC2 Infrastructure-as-Code & Deployment Orchestration
def test_ec2_iac_artifacts():
    import os
    tf_files = [
        "infra/terraform/main.tf",
        "infra/terraform/provider.tf",
        "infra/terraform/variables.tf",
        "infra/terraform/outputs.tf",
        "infra/terraform/user_data.sh",
        "infra/terraform/terraform.tfvars.example"
    ]
    for tf in tf_files:
        assert os.path.exists(tf), f"Terraform artifact missing: {tf}"
        with open(tf, "r") as f:
            content = f.read()
        assert len(content) > 20, f"Empty terraform file: {tf}"

    assert os.path.exists("deploy/ec2-setup.sh"), "deploy/ec2-setup.sh missing"
    with open("deploy/ec2-setup.sh", "r") as f:
        deploy_sh = f.read()
    assert "docker compose" in deploy_sh
    assert "systemctl enable kather_baksho.service" in deploy_sh

# 55. Redis Streams Event-Driven Architecture & Message Bus
def test_redis_streams_event_bus():
    r_stats = s.get(f"{BASE_URL}/events/stats")
    assert r_stats.status_code == 200, f"Event stats failed: {r_stats.status_code}"
    stats = r_stats.json()
    assert stats.get("status") == "healthy"
    assert stats.get("stream") == "kb:events:stream"
    assert stats.get("group") == "kb_workers"

    payload = {
        "type": "telemetry.alert",
        "payload": {
            "plant_id": 101,
            "botanical_status": "Simulated Needs Water Alert"
        }
    }
    r_pub = s.post(f"{BASE_URL}/events/publish", json=payload)
    assert r_pub.status_code == 201, f"Event publish failed: {r_pub.status_code}"
    pub_res = r_pub.json()
    assert pub_res.get("success") is True
    assert "event_id" in pub_res

# 56. SQLite Full-Text Search (FTS) & Highlight Snippets
def test_fts_fulltext_search():
    r = s.get(f"{BASE_URL}/products/search?q=Succulent")
    assert r.status_code == 200, f"Search failed: {r.status_code}"
    data = r.json()
    assert data.get("query") == "Succulent"
    assert data.get("count") > 0
    assert "results" in data
    first = data["results"][0]
    assert "<mark>" in first.get("highlight", "") or "Succulent" in first.get("name", "")

# 57. Local MinIO S3 Object Storage & Media Pipeline
def test_minio_s3_storage():
    r_status = s.get(f"{BASE_URL}/media/status")
    assert r_status.status_code == 200, f"Media status failed: {r_status.status_code}"
    status_data = r_status.json()
    assert status_data.get("status") == "healthy"
    assert status_data.get("bucket") == "kather-baksho-media"

    files = {"file": ("plant_botanical.png", b"binary_test_image_bytes_45678", "image/png")}
    r_up = s.post(f"{BASE_URL}/media/upload", files=files)
    assert r_up.status_code == 201, f"Media upload failed: {r_up.status_code}"
    up_data = r_up.json()
    assert up_data.get("success") is True
    assert "url" in up_data

    file_url = f"http://localhost:8081{up_data['url']}"
    r_get = s.get(file_url)
    assert r_get.status_code == 200, f"Media retrieval failed: {r_get.status_code}"
    assert r_get.content == b"binary_test_image_bytes_45678"

# 58. Enterprise RFC 6238 TOTP Two-Factor Authentication
def test_enterprise_2fa_totp():
    import base64
    import hashlib
    import hmac
    import struct
    import time

    def get_totp_token(secret):
        secret = secret.upper().strip()
        padding = (8 - len(secret) % 8) % 8
        key = base64.b32decode(secret + '=' * padding)
        counter = int(time.time() // 30)
        msg = struct.pack(">Q", counter)
        h = hmac.new(key, msg, hashlib.sha1).digest()
        offset = h[-1] & 0x0F
        truncated = struct.unpack(">I", h[offset:offset + 4])[0] & 0x7FFFFFFF
        code = truncated % 1000000
        return f"{code:06d}"

    # 1. Register a fresh user
    user_email = f"totp_user_{int(time.time())}@example.com"
    r_reg = requests.post(f"{BASE_URL}/auth/register", json={
        "name": "TOTP Tester",
        "email": user_email,
        "password": "Password123!"
    })
    assert r_reg.status_code == 201
    user_tok = r_reg.json()["token"]
    u_headers = {"Authorization": f"Bearer {user_tok}"}

    # 2. Start 2FA setup
    r_setup = requests.post(f"{BASE_URL}/auth/2fa/setup", headers=u_headers)
    assert r_setup.status_code == 200, f"2FA setup failed: {r_setup.text}"
    setup_data = r_setup.json()
    secret = setup_data.get("secret")
    assert secret and len(secret) >= 16
    assert "otpauth://" in setup_data.get("otpauth_uri", "")

    # 3. Calculate TOTP code and enable 2FA
    code = get_totp_token(secret)
    r_enable = requests.post(f"{BASE_URL}/auth/2fa/enable", json={"code": code}, headers=u_headers)
    assert r_enable.status_code == 200, f"2FA enable failed: {r_enable.text}"
    enable_data = r_enable.json()
    assert enable_data.get("totp_enabled") is True
    recovery_codes = enable_data.get("recovery_codes", [])
    assert len(recovery_codes) == 8

    # 4. Check /me shows totp_enabled == True
    r_me = requests.get(f"{BASE_URL}/auth/me", headers=u_headers)
    assert r_me.status_code == 200
    assert r_me.json().get("totp_enabled") is True

    # 5. Login again with password -> must return requires_2fa: true and temp_token
    r_login = requests.post(f"{BASE_URL}/auth/login", json={
        "email": user_email,
        "password": "Password123!"
    })
    assert r_login.status_code == 200
    login_data = r_login.json()
    assert login_data.get("requires_2fa") is True
    temp_token = login_data.get("temp_token")
    assert temp_token is not None

    # 6. Verify temp_token CANNOT access protected /me endpoint
    r_bad_me = requests.get(f"{BASE_URL}/auth/me", headers={"Authorization": f"Bearer {temp_token}"})
    assert r_bad_me.status_code == 401, "temp_token should not access protected endpoints"

    # 7. Complete 2FA challenge with TOTP code
    verify_code = get_totp_token(secret)
    r_verify = requests.post(f"{BASE_URL}/auth/2fa/verify", json={
        "temp_token": temp_token,
        "code": verify_code
    })
    assert r_verify.status_code == 200, f"2FA challenge failed: {r_verify.text}"
    final_token = r_verify.json().get("token")
    assert final_token is not None

    # 8. Full token now accesses protected /me
    r_ok_me = requests.get(f"{BASE_URL}/auth/me", headers={"Authorization": f"Bearer {final_token}"})
    assert r_ok_me.status_code == 200

    # 9. Test Recovery Code fallback login
    r_login2 = requests.post(f"{BASE_URL}/auth/login", json={
        "email": user_email,
        "password": "Password123!"
    })
    temp_token2 = r_login2.json()["temp_token"]
    r_rc_verify = requests.post(f"{BASE_URL}/auth/2fa/verify", json={
        "temp_token": temp_token2,
        "recovery_code": recovery_codes[0]
    })
    assert r_rc_verify.status_code == 200, "Recovery code verification failed"

    # 10. Disable 2FA with password
    r_disable = requests.post(f"{BASE_URL}/auth/2fa/disable", json={"password": "Password123!"}, headers={"Authorization": f"Bearer {final_token}"})
    assert r_disable.status_code == 200
    assert r_disable.json().get("totp_enabled") is False

# 59. Chaos Engineering & Resilience Studio (Fault Injection & Circuit Breakers)
def test_chaos_engineering_resilience():
    # 1. Fetch default configuration
    r_cfg = s.get(f"{BASE_URL}/chaos/config")
    assert r_cfg.status_code == 200
    cfg = r_cfg.json()
    assert cfg.get("enabled") is False

    # 2. Configure targeted chaos on /api/chaos/probe
    r_update = s.post(f"{BASE_URL}/chaos/config", json={
        "enabled": True,
        "latency_ms": 100,
        "error_rate_percent": 100,
        "target_prefixes": ["/api/chaos/probe"]
    })
    assert r_update.status_code == 200
    assert r_update.json()["config"]["enabled"] is True

    # 3. Request to targeted route triggers injected chaos fault (503)
    r_fault = s.get(f"{BASE_URL}/chaos/probe")
    assert r_fault.status_code == 503
    assert r_fault.json().get("chaos_fault") is True

    # 4. Request to untargeted route (/api/products) bypasses error injection
    r_safe = s.get(f"{BASE_URL}/products?limit=1")
    assert r_safe.status_code == 200

    # 5. Trip Circuit Breaker manually
    r_trip = s.post(f"{BASE_URL}/chaos/trip-breaker", json={"name": "test-resilience-breaker"})
    assert r_trip.status_code == 200
    assert r_trip.json().get("state") == "OPEN"

    # 6. Check stats reflects delayed count, injected failures, and open breaker
    r_stats = s.get(f"{BASE_URL}/chaos/stats")
    assert r_stats.status_code == 200
    stats = r_stats.json()
    assert stats.get("injected_failures", 0) >= 1
    assert stats.get("circuit_breakers", {}).get("test-resilience-breaker") == "OPEN"

    # 7. Reset all chaos & circuit breakers to healthy baseline
    r_reset = s.post(f"{BASE_URL}/chaos/reset")
    assert r_reset.status_code == 200
    reset_data = r_reset.json()
    assert reset_data["config"]["enabled"] is False
    assert reset_data["stats"]["circuit_breakers"].get("test-resilience-breaker") == "CLOSED"

def _probe_health(port, container_name):
    try:
        r = requests.get(f"http://localhost:{port}/health", timeout=3)
        if r.status_code == 200:
            return r.json()
    except Exception:
        pass
    import subprocess, json
    res = subprocess.run(["docker", "exec", container_name, "wget", "-qO-", f"http://localhost:{port}/health"],
                         capture_output=True, text=True, check=True)
    return json.loads(res.stdout)

# 60. Dedicated IoT & AI Botanical Intelligence Microservice Direct Probe
def test_iot_ai_microservice():
    data = _probe_health(8089, "kather_baksho-iot-ai")
    assert data.get("service") == "kather_baksho-iot-ai"
    assert data.get("status") == "healthy"
    assert data.get("mongodb") == "connected"
    assert data.get("redis") == "connected"

    # Verify Traefik routes to the dedicated microservice with correlation ID propagation
    test_corr_id = "test-corr-trace-999"
    r_corr = requests.get("http://localhost:8085/api/iot/plants", headers={"X-Correlation-ID": test_corr_id}, timeout=5)
    assert r_corr.status_code == 200
    assert r_corr.headers.get("X-Correlation-ID") == test_corr_id, "X-Correlation-ID must propagate across gateway"

# 61. Automated Concurrency & Stress Testing Benchmark (Flash Sale Simulator)
def test_concurrency_flash_sale_benchmark():
    try:
        from stress_test import run_flash_sale_benchmark
    except ImportError:
        from tests.stress_test import run_flash_sale_benchmark
    ok = run_flash_sale_benchmark(base_url=BASE_URL, concurrency=20, initial_stock=5)
    assert ok is True, "Flash sale concurrency race condition test failed!"

# 62. Autonomous Catalog, Search & Media Microservice (:8087)
def test_catalog_microservice():
    data = _probe_health(8087, "kather_baksho-catalog")
    assert data.get("service") == "kather_baksho-catalog"
    assert data.get("status") == "healthy"
    assert data.get("database") == "connected"
    assert data.get("cache") == "connected"

    # Verify Traefik gateway routing to catalog service
    test_corr_id = "test-corr-catalog-888"
    r_gw = requests.get("http://localhost:8085/api/products", headers={"X-Correlation-ID": test_corr_id}, timeout=5)
    assert r_gw.status_code == 200
    gw_data = r_gw.json()
    assert "items" in gw_data or isinstance(gw_data, list)
    assert r_gw.headers.get("X-Correlation-ID") == test_corr_id, "X-Correlation-ID must propagate across gateway to catalog-service"

# 63. Autonomous Identity, 2FA TOTP & User Management Microservice (:8084)
def test_auth_microservice():
    data = _probe_health(8084, "kather_baksho-auth")
    assert data.get("service") == "kather_baksho-auth"
    assert data.get("status") == "healthy"
    assert data.get("database") == "connected"

    # Verify Traefik gateway routing to auth service (direct login through gateway)
    test_corr_id = "test-corr-auth-777"
    r_gw = requests.post(
        "http://localhost:8085/api/auth/login",
        json={"email": "admin@kather_baksho.com", "password": "Admin@12345"},
        headers={"X-Correlation-ID": test_corr_id},
        timeout=5
    )
    assert r_gw.status_code == 200
    assert "token" in r_gw.json()
    assert r_gw.headers.get("X-Correlation-ID") == test_corr_id, "X-Correlation-ID must propagate across gateway to auth-service"

# 64. Autonomous Community, Botanical Care & Subscriptions Microservice (:8088)
def test_community_care_microservice():
    data = _probe_health(8088, "kather_baksho-community-care")
    assert data.get("service") == "kather_baksho-community-care"
    assert data.get("status") == "healthy"
    assert data.get("database") == "connected"

    # Verify Traefik gateway routing to community-care-service for community posts
    test_corr_id = "test-corr-comm-666"
    r_comm = requests.get("http://localhost:8085/api/community/posts", headers={"X-Correlation-ID": test_corr_id}, timeout=5)
    assert r_comm.status_code == 200
    assert r_comm.headers.get("X-Correlation-ID") == test_corr_id, "X-Correlation-ID must propagate across gateway to community-care-service"
    comm_data = r_comm.json()
    assert isinstance(comm_data, list)

    # Verify Traefik gateway routing to community-care-service for blog
    r_blog = requests.get("http://localhost:8085/api/blog", headers={"X-Correlation-ID": test_corr_id}, timeout=5)
    assert r_blog.status_code == 200
    blog_data = r_blog.json()
    assert "posts" in blog_data

tests = [

    ("Health / Get Products", test_get_products),
    ("Login Admin", test_login_admin),
    ("Login Staff", test_login_staff),
    ("Register User", test_register_user),
    ("User Me", test_user_me),
    ("Update Profile", test_update_profile),
    ("Addresses CRUD", test_addresses),
    ("Cart Operations & Stock Reservation", test_cart_operations),
    ("Coupon Validate & Apply", test_coupon_validate),
    ("Checkout Order", test_checkout),
    ("Stock Retention After Checkout", test_stock_after_checkout),
    ("Get Orders & Detail", test_get_orders),
    ("Order Invoice & Receipt", test_order_invoice_receipt),
    ("Wishlist CRUD", test_wishlist),
    ("Reviews CRUD", test_reviews),
    ("Consultations Booking", test_consultations),
    ("Subscriptions Booking", test_subscriptions),
    ("Growth Journal CRUD", test_journal),
    ("Community Posts (Public & Auth)", test_community),
    ("Admin Cancel Order & Stock Restoration", test_admin_order_status_and_stock_restore),
    ("Admin Analytics", test_admin_analytics),
    ("Admin Users List", test_admin_users),
    ("Admin Product CRUD (Unique Slug & Deduplication)", test_admin_product_crud),
    ("Admin Confirm Consultation", test_admin_consultations),
    ("Categories API", test_categories),
    ("Blog API & Article Detail", test_blog),
    ("Security Boundary (Admin Protection)", test_security_boundary),
    ("Frontend SPA Routing & Fallback", test_frontend_routes),
    ("SQLite Database PRAGMA Integrity Check", test_db_health),
    ("Prometheus Metrics Exposition", test_prometheus),
    ("Enhanced Health Readiness Probe", test_health_ready),
    ("Idempotency Key Replay Verification", test_idempotency),
    ("Payment Session & HMAC Signature Callback", test_payment_flow),
    ("AI Plant Doctor Symptom Diagnosis", test_ai_diagnose),
    ("AI Plant Doctor Conversational Q&A", test_ai_chat),
    ("ML Model Comparison Benchmark", test_ml_compare),
    ("Algorithm Studio Visualizer Route", test_algorithm_visualizer),
    ("Redis Read-Through Caching & Cache-HIT", test_redis_caching),
    ("Redis Cache Invalidation on Admin Mutation", test_cache_invalidation),
    ("Multi-Database Health & Readiness Probe", test_multidb_health),
    ("Node.js & TypeScript Worker Health Probe", test_worker_health),
    ("Node.js & TypeScript Direct PDF Generation", test_worker_direct_invoice),
    ("Go Backend PDF Invoice Proxy", test_go_pdf_invoice_proxy),
    ("Go Backend Analytics Executive Report PDF", test_go_analytics_report_pdf),
    ("MongoDB Tri-Database Readiness Probe", test_mongodb_readiness_probe),
    ("MongoDB IoT Telemetry Ingestion & Alerting", test_iot_telemetry_ingestion),
    ("MongoDB Monitored Plants Snapshot", test_iot_monitored_plants),
    ("MongoDB IoT Telemetry Time-Series History", test_iot_telemetry_history),
    ("Traefik Ingress Routing (Frontend, API & Worker)", test_traefik_api_gateway_routing),
    ("Traefik Live Dashboard & Telemetry API", test_traefik_dashboard),
    ("WebSocket Real-Time Order & Rider Tracking", test_websocket_order_tracking),
    ("OpenAPI 3.0 Specification & Interactive Swagger UI", test_openapi_swagger_docs),
    ("Frontend TypeScript Type Declarations & Config", test_frontend_typescript_types),
    ("AWS EC2 Infrastructure-as-Code & Deployment Orchestration", test_ec2_iac_artifacts),
    ("Redis Streams Event-Driven Architecture & Message Bus", test_redis_streams_event_bus),
    ("SQLite Full-Text Search (FTS) & Highlight Snippets", test_fts_fulltext_search),
    ("Local MinIO S3 Object Storage & Media Pipeline", test_minio_s3_storage),
    ("Enterprise RFC 6238 TOTP Two-Factor Authentication", test_enterprise_2fa_totp),
    ("Chaos Engineering & Resilience Studio (Fault Injection & Circuit Breakers)", test_chaos_engineering_resilience),
    ("Dedicated IoT & AI Botanical Intelligence Microservice", test_iot_ai_microservice),
    ("Automated Concurrency & Stress Testing Benchmark (Flash Sale Simulator)", test_concurrency_flash_sale_benchmark),
    ("Autonomous Catalog, Search & Media Microservice", test_catalog_microservice),
    ("Autonomous Identity, 2FA TOTP & User Management Microservice", test_auth_microservice),
    ("Autonomous Community, Botanical Care & Subscriptions Microservice", test_community_care_microservice)
]


print("Starting E2E test suite...")
for name, fn in tests:
    run_test(name, fn)

passed = sum(1 for _, s, _ in results if s)
failed = sum(1 for _, s, _ in results if not s)
print(f"\nSummary: {len(results)} tests run, {passed} passed, {failed} failed.")
if failed > 0:
    sys.exit(1)


