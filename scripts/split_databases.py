#!/usr/bin/env python3
"""
Database Splitting & Provisioning Script for Kather Baksho Microservices.
Splits a monolithic SQLite database into 4 isolated, private domain databases:
  - auth.db        (Identity & Access Management)
  - catalog.db     (Products, Categories, Reviews, Wishlists)
  - community.db   (Community, Botanical Care, Journals, Consultations)
  - orders.db      (Orders, Carts, Checkout, Fulfillment, Loyalty)
"""

import os
import sys
import sqlite3
import argparse

DOMAIN_TABLES = {
    "auth.db": [
        "users",
        "addresses",
        "email_verifications",
    ],
    "catalog.db": [
        "products",
        "categories",
        "reviews",
        "wishlist_items",
        "page_views",
    ],
    "community.db": [
        "community_posts",
        "community_comments",
        "community_likes",
        "community_follows",
        "community_bookmarks",
        "community_groups",
        "community_group_members",
        "community_questions",
        "community_answers",
        "growth_journals",
        "care_schedules",
        "blog_posts",
        "consultations",
        "subscriptions",
        "subscription_deliveries",
        "notifications",
    ],
    "orders.db": [
        "orders",
        "order_items",
        "carts",
        "cart_items",
        "coupons",
        "care_reminders",
        "corporate_quotes",
        "gift_cards",
        "return_requests",
        "idempotency_records",
        "user_memberships",
        "referral_codes",
        "order_events",
        "achievements",
        "user_achievements",
        "referrals",
        "membership_tiers",
        "coupon_rewards",
        "corporate_orders",
        "guest_orders",
        "shipping_rules",
        "tax_rules",
    ],
}

def split_database(source_path: str, output_dir: str):
    print(f"[*] Opening source SQLite database: {source_path}")
    if not os.path.isfile(source_path):
        print(f"[!] Error: Source database not found: {source_path}")
        sys.exit(1)

    os.makedirs(output_dir, exist_ok=True)
    src_conn = sqlite3.connect(source_path)
    src_cur = src_conn.cursor()

    total_migrated_tables = 0
    total_migrated_rows = 0

    for db_filename, tables in DOMAIN_TABLES.items():
        target_path = os.path.join(output_dir, db_filename)
        # Remove target file if exists to start fresh and clean
        if os.path.exists(target_path):
            os.remove(target_path)
            for suffix in ["-wal", "-shm"]:
                if os.path.exists(target_path + suffix):
                    os.remove(target_path + suffix)

        print(f"\n[+] Provisioning {db_filename} -> {target_path}")
        dst_conn = sqlite3.connect(target_path)
        dst_cur = dst_conn.cursor()
        dst_conn.execute("PRAGMA journal_mode = WAL;")

        for tbl in tables:
            # Check if table exists in source
            src_cur.execute("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", (tbl,))
            row = src_cur.fetchone()
            if not row:
                print(f"    - Notice: Table '{tbl}' does not exist in source, skipping.")
                continue

            table_sql = row[0]
            dst_cur.execute(table_sql)

            # Copy all records
            src_cur.execute(f'SELECT * FROM "{tbl}"')
            rows = src_cur.fetchall()
            if rows:
                placeholders = ",".join(["?"] * len(rows[0]))
                dst_cur.executemany(f'INSERT INTO "{tbl}" VALUES ({placeholders})', rows)

            # Copy indexes
            src_cur.execute("SELECT sql FROM sqlite_master WHERE type='index' AND tbl_name=? AND sql IS NOT NULL", (tbl,))
            index_rows = src_cur.fetchall()
            for idx_row in index_rows:
                idx_sql = idx_row[0]
                try:
                    dst_cur.execute(idx_sql)
                except Exception as e:
                    print(f"    - Warning on index '{idx_sql}': {e}")

            dst_conn.commit()

            # Verify count
            dst_cur.execute(f'SELECT COUNT(*) FROM "{tbl}"')
            cnt = dst_cur.fetchone()[0]
            print(f"    ✓ {tbl}: {cnt} rows copied and verified")
            total_migrated_tables += 1
            total_migrated_rows += cnt

        dst_conn.close()

    src_conn.close()
    print(f"\n[SUCCESS] Split completed: {total_migrated_tables} tables, {total_migrated_rows} rows preserved.")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Split monolithic database into isolated microservice databases")
    parser.add_argument("--source", default="/tmp/docker_kather_baksho.db", help="Path to source SQLite database")
    parser.add_argument("--output-dir", default="/tmp/split_dbs", help="Directory to write isolated databases")
    args = parser.parse_args()

    split_database(args.source, args.output_dir)

