// seed-demo.mjs — top up the SQLite demo database with a large, realistic
// dataset WITHOUT needing the Go toolchain. Talks straight to katherbox.db
// via Node's built-in sqlite module (Node >= 22).
//
//   node scripts/seed-demo.mjs            # from backend/, default targets
//   node scripts/seed-demo.mjs --users 120 --orders 250
//
// Idempotent: seeded users use "@katherbox.demo" emails and are only created
// up to the target; orders/reviews/addresses are only added when the target
// count isn't met yet. All seeded customer accounts share the password of the
// existing customer@test.com account:  Customer@12345
//
// This is the "make the demo work right now" path. cmd/seedusers +
// cmd/seeddummy (Go) are the reproducible equivalents once Go is available.

import { DatabaseSync } from "node:sqlite";
import { fileURLToPath } from "node:url";
import { copyFileSync, existsSync } from "node:fs";
import path from "node:path";

const argv = process.argv.slice(2);
const opt = (name, def) => {
  const i = argv.indexOf("--" + name);
  return i >= 0 && argv[i + 1] ? Number(argv[i + 1]) : def;
};
const TARGET_USERS = opt("users", 90); // total users to reach
const TARGET_ORDERS = opt("orders", 220); // total orders to reach
const TARGET_REVIEWS = opt("reviews", 180);
const TARGET_ADDRESSES = opt("addresses", 90);

const dbPath = path.resolve(fileURLToPath(new URL(".", import.meta.url)), "..", "katherbox.db");

// One-time safety copy before the first top-up.
const backup = dbPath + ".bak-preseed";
if (!existsSync(backup)) {
  copyFileSync(dbPath, backup);
  console.log(`(backed up ${path.basename(dbPath)} -> ${path.basename(backup)})`);
}

const db = new DatabaseSync(dbPath);

// ---------- helpers ----------
const rnd = (n) => Math.floor(Math.random() * n);
const pick = (a) => a[rnd(a.length)];
const chance = (p) => Math.random() < p;
const pad = (n, w = 2) => String(n).padStart(w, "0");

function daysAgo(d) {
  const t = new Date(Date.now() - d * 86400_000 - rnd(86400) * 1000);
  return `${t.getFullYear()}-${pad(t.getMonth() + 1)}-${pad(t.getDate())} ${pad(t.getHours())}:${pad(t.getMinutes())}:${pad(t.getSeconds())}`;
}

const FIRST = ["Aarav","Sara","Rafi","Maya","Imran","Nadia","Tariq","Lamia","Ravi","Anika","Sumi","Karan","Priya","Hasan","Rina","Omar","Faria","Bilal","Tania","Sabbir","Mehjabin","Asif","Tahsin","Rownak","Nazia","Sadman","Aditi","Taseen","Mahmood","Zara","Nabil","Ishrat","Fahim","Sneha","Arif","Munia","Shakib","Tisha","Rakib","Prova","Jamil","Ruma","Kamal","Shirin","Emon"];
const LAST = ["Khan","Rahman","Hossain","Akter","Islam","Chowdhury","Ahmed","Begum","Sultana","Miah","Das","Roy","Sarkar","Talukder","Bhuiyan","Haque","Sheikh","Biswas","Saha","Karim"];
const CITIES = ["Dhaka","Chattogram","Sylhet","Rajshahi","Khulna","Barishal","Rangpur","Mymensingh"];
const NOTES = ["","","Leave with the guard","Call on arrival","Handle with care — fragile pots","Deliver after 5pm"];
const PAY = ["bkash","nagad","rocket","card","cod"];
const REVIEW_TEXT = ["Arrived in perfect condition, leaves are vibrant.","Healthy plant, a little smaller than expected but good value.","Beautiful pot and well packaged. Will buy again.","Already a new leaf in two weeks!","Soil was slightly dry on arrival but the plant is fine.","Better than the photo — very happy.","Customer service replaced a damaged leaf, great support.","Exactly as described, strong roots.","Survived my beginner mistakes. Forgiving plant.","Pot matches my decor perfectly.","Five stars, ordered two more.","Lush and full, highly recommended."];
const REM_TYPES = ["watering","fertilizer","repotting","pruning","mist","rotate"];

const phone = () => `+8801${3 + rnd(7)}${pad(rnd(10_000_000), 7)}`;
const addrLine = () => `House ${rnd(90) + 1}, Road ${rnd(30) + 1}`;

// ---------- reference data ----------
const CUST_HASH =
  db.prepare("SELECT password FROM users WHERE email = 'customer@test.com'").get()?.password ||
  db.prepare("SELECT password FROM users WHERE role='customer' OR role='staff' LIMIT 1").get()?.password;
if (!CUST_HASH) throw new Error("no existing user to borrow a password hash from");

const products = db
  .prepare("SELECT id, name, price, category FROM products WHERE deleted_at IS NULL AND price > 0")
  .all();
if (products.length === 0) throw new Error("no products in db — run cmd/seedproducts first");

const now = () => daysAgo(0);

const tx = db.prepare("BEGIN");
const commit = db.prepare("COMMIT");

// ---------- 1. users ----------
tx.run();
const insUser = db.prepare(
  `INSERT INTO users (created_at, updated_at, name, email, password, role, points, email_verified, phone, address)
   VALUES (?,?,?,?,?,?,?,?,?,?)`
);
let usersHave = db.prepare("SELECT COUNT(*) n FROM users").get().n;
let usersMade = 0;
const seenEmail = new Set(db.prepare("SELECT email FROM users").all().map((r) => r.email));
while (usersHave + usersMade < TARGET_USERS) {
  const fn = pick(FIRST), ln = pick(LAST);
  let email = `${fn}.${ln}`.toLowerCase() + "@katherbox.demo";
  let k = 1;
  while (seenEmail.has(email)) email = `${fn}.${ln}${k++}`.toLowerCase() + "@katherbox.demo";
  seenEmail.add(email);
  const created = daysAgo(rnd(365) + 1);
  // Bulk demo users are all shoppers. "staff" is an operational role assigned
  // deliberately (see cmd/resetusers / cmd/seedusers), not sprinkled at random.
  insUser.run(
    created, created, `${fn} ${ln}`, email, CUST_HASH, "customer",
    rnd(20) * 50, chance(0.85) ? 1 : 0, phone(),
    `${addrLine()}, ${pick(CITIES)} ${pad(rnd(9000) + 1000, 4)}, Bangladesh`
  );
  usersMade++;
}
commit.run();
console.log(`users:     +${usersMade}  (total ${usersHave + usersMade})`);

// all customer/staff ids for later fan-out
const customerIds = db
  .prepare("SELECT id, name FROM users WHERE role IN ('customer','staff') AND deleted_at IS NULL")
  .all();

// ---------- 2. addresses ----------
tx.run();
const insAddr = db.prepare(
  `INSERT INTO addresses (created_at, updated_at, user_id, label, recipient, phone, line1, line2, city, region, postal_code, country, is_default)
   VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
);
let addrHave = db.prepare("SELECT COUNT(*) n FROM addresses").get().n;
let addrMade = 0;
for (const u of customerIds) {
  if (addrHave + addrMade >= TARGET_ADDRESSES) break;
  const has = db.prepare("SELECT COUNT(*) n FROM addresses WHERE user_id = ?").get(u.id).n;
  if (has > 0) continue;
  const count = 1 + rnd(2);
  for (let i = 0; i < count; i++) {
    const city = pick(CITIES);
    insAddr.run(
      now(), now(), u.id, pick(["Home", "Office", "Parents'"]), u.name, phone(),
      addrLine(), pick(["", "Flat 3B", "2nd floor", "Apt 5"]),
      city, city + " Division", pad(rnd(9000) + 1000, 4), "Bangladesh", i === 0 ? 1 : 0
    );
    addrMade++;
  }
}
commit.run();
console.log(`addresses: +${addrMade}  (total ${addrHave + addrMade})`);

// ---------- 3. orders + order_items ----------
tx.run();
const insOrder = db.prepare(
  `INSERT INTO orders (created_at, updated_at, user_id, total_price, status, gift_wrap, payment_method, payment_status, transaction_id, coupon_code, discount_amount)
   VALUES (?,?,?,?,?,?,?,?,?,?,?)`
);
const insItem = db.prepare(
  `INSERT INTO order_items (created_at, updated_at, order_id, product_id, quantity, price) VALUES (?,?,?,?,?,?)`
);
let ordersHave = db.prepare("SELECT COUNT(*) n FROM orders").get().n;
let ordersMade = 0, itemsMade = 0;
const stageFor = (d) =>
  d > 25 ? pick(["Delivered", "Delivered", "Delivered", "Returned"]) :
  d > 12 ? pick(["Delivered", "On the Way", "Packed"]) :
  d > 4 ? pick(["On the Way", "Packed", "Processing"]) :
  pick(["Processing", "Pending", "Pending"]);

while (ordersHave + ordersMade < TARGET_ORDERS) {
  const u = pick(customerIds);
  const d = rnd(150);
  const when = daysAgo(d);
  const status = stageFor(d);
  const lineCount = 1 + rnd(4);
  const lines = [];
  let subtotal = 0;
  const used = new Set();
  for (let i = 0; i < lineCount; i++) {
    const p = pick(products);
    if (used.has(p.id)) continue;
    used.add(p.id);
    const qty = 1 + rnd(3);
    lines.push({ id: p.id, qty, price: p.price });
    subtotal += p.price * qty;
  }
  if (lines.length === 0) continue;

  let discount = 0, coupon = "";
  if (chance(0.3)) {
    const pct = pick([5, 10, 15, 20]);
    discount = Math.round(subtotal * pct) / 100;
    coupon = `KB-SAVE-${pct}`;
  }
  const gift = chance(0.35) ? 1 : 0;
  const shipping = subtotal - discount < 1500 ? 60 : 0;
  const total = Math.round((subtotal - discount + (gift ? 50 : 0) + shipping) * 100) / 100;
  const method = pick(PAY);
  let payStatus = method === "cod" ? "Pending COD" : "Paid";
  if (status === "Delivered") payStatus = "Paid";

  const r = insOrder.run(
    when, when, u.id, total, status, gift, method, payStatus,
    `TRX${pad(rnd(99_999_999), 8)}`, coupon, discount
  );
  const oid = r.lastInsertRowid;
  for (const l of lines) {
    insItem.run(when, when, oid, l.id, l.qty, l.price);
    itemsMade++;
  }
  ordersMade++;
}
commit.run();
console.log(`orders:    +${ordersMade}  (total ${ordersHave + ordersMade}, +${itemsMade} items)`);

// ---------- 4. reviews (respect unique (user_id, product_id)) ----------
tx.run();
const insReview = db.prepare(
  `INSERT OR IGNORE INTO reviews (created_at, updated_at, user_id, product_id, rating, comment, user_name) VALUES (?,?,?,?,?,?,?)`
);
let reviewsHave = db.prepare("SELECT COUNT(*) n FROM reviews").get().n;
let reviewsMade = 0, tries = 0;
while (reviewsHave + reviewsMade < TARGET_REVIEWS && tries < TARGET_REVIEWS * 6) {
  tries++;
  const u = pick(customerIds);
  const p = pick(products);
  const when = daysAgo(rnd(200));
  const res = insReview.run(
    when, when, u.id, p.id,
    chance(0.75) ? 5 : 4, pick(REVIEW_TEXT), u.name
  );
  if (res.changes > 0) reviewsMade++;
}
commit.run();
console.log(`reviews:   +${reviewsMade}  (total ${reviewsHave + reviewsMade})`);

// ---------- 5. care reminders + notifications ----------
tx.run();
const insRem = db.prepare(
  `INSERT INTO care_reminders (created_at, updated_at, user_id, product_id, type, next_due_date, interval_days, completed) VALUES (?,?,?,?,?,?,?,?)`
);
const insNote = db.prepare(
  `INSERT INTO notifications (created_at, updated_at, user_id, message, type, is_read) VALUES (?,?,?,?,?,?)`
);
let remMade = 0, noteMade = 0;
for (const u of customerIds) {
  if (chance(0.4)) {
    const p = pick(products);
    const due = new Date(Date.now() + (rnd(30) - 5) * 86400_000);
    insRem.run(now(), now(), u.id, p.id, pick(REM_TYPES),
      `${due.getFullYear()}-${pad(due.getMonth() + 1)}-${pad(due.getDate())}`,
      pick([3, 7, 14, 21, 30]), chance(0.3) ? 1 : 0);
    remMade++;
  }
  if (chance(0.5)) {
    insNote.run(now(), now(), u.id,
      pick(["Your order has shipped 🚚", "New care guide for your plants 🌱", "Green Points added to your account ✨", "A plant on your wishlist is back in stock", "Your subscription box ships next week 📦"]),
      pick(["order", "system", "loyalty", "wishlist"]), chance(0.4) ? 1 : 0);
    noteMade++;
  }
}
commit.run();
console.log(`reminders: +${remMade}   notifications: +${noteMade}`);

// ---------- snapshot ----------
console.log("\n--- row counts ---");
for (const t of ["users", "addresses", "orders", "order_items", "reviews", "products", "subscriptions", "consultations", "return_requests", "care_reminders", "notifications", "blog_posts", "coupons", "gift_cards", "community_questions"]) {
  console.log(`  ${t.padEnd(18)} ${db.prepare("SELECT COUNT(*) n FROM " + t).get().n}`);
}
db.close();
console.log("\nDone. Seeded customers log in with:  <name>@katherbox.demo  /  Customer@12345");
