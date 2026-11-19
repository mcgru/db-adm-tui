/*
 * Idempotent MongoDB test fixture.
 *
 * Run through mongo/apply.sh as the root user. It is safe to rerun: existing
 * users/roles are updated instead of recreated and collections are seeded only
 * when empty.
 */

const admin = db.getSiblingDB("admin");

const env = (name, fallback) => process.env[name] || fallback;

function exists(getter) {
  try {
    return getter() !== null;
  } catch (e) {
    return false;
  }
}

function ensureUser(dbName, user, password, roles) {
  const target = db.getSiblingDB(dbName);
  if (exists(() => target.getUser(user))) {
    target.updateUser(user, { pwd: password, roles: roles });
    print(`updated user ${dbName}.${user}`);
  } else {
    target.createUser({ user: user, pwd: password, roles: roles });
    print(`created user ${dbName}.${user}`);
  }
}

function ensureRole(dbName, role, privileges, roles) {
  const target = db.getSiblingDB(dbName);
  const inherited = roles || [];
  if (exists(() => target.getRole(role))) {
    target.updateRole(role, { privileges: privileges, roles: inherited });
    print(`updated role ${dbName}.${role}`);
  } else {
    target.createRole({ role: role, privileges: privileges, roles: inherited });
    print(`created role ${dbName}.${role}`);
  }
}

function ensureCollection(dbName, collectionName) {
  const target = db.getSiblingDB(dbName);
  if (!target.getCollectionNames().includes(collectionName)) {
    target.createCollection(collectionName);
  }
  return target.getCollection(collectionName);
}

function seed(collection, documents) {
  if (collection.countDocuments({}) === 0) {
    collection.insertMany(documents);
    print(`seeded ${collection.getFullName()} with ${documents.length} documents`);
  }
}

const adminUser = env("MONGO_ADMIN_USERNAME", "mongo_admin");
const appUser = env("MONGO_APP_USERNAME", "app_user");
const analystUser = env("MONGO_ANALYST_USERNAME", "analyst");

ensureUser("admin", adminUser, env("MONGO_ADMIN_PASSWORD"), [
  { role: "userAdminAnyDatabase", db: "admin" },
  { role: "dbAdminAnyDatabase", db: "admin" },
  { role: "readWriteAnyDatabase", db: "admin" },
  { role: "clusterMonitor", db: "admin" }
]);

ensureRole("analytics", "analytics_read", [
  {
    resource: { db: "analytics", collection: "" },
    actions: ["find", "listCollections"]
  }
], []);

ensureRole("appdb", "app_writer", [
  {
    resource: { db: "appdb", collection: "customers" },
    actions: ["find", "insert", "update"]
  },
  {
    resource: { db: "appdb", collection: "orders" },
    actions: ["find", "insert", "update"]
  }
], []);

ensureRole("sandbox", "sandbox_reader", [
  {
    resource: { db: "sandbox", collection: "" },
    actions: ["find", "listCollections"]
  }
], []);

ensureUser("appdb", appUser, env("MONGO_APP_PASSWORD"), [
  { role: "app_writer", db: "appdb" },
  { role: "readWrite", db: "appdb" }
]);

ensureUser("analytics", analystUser, env("MONGO_ANALYST_PASSWORD"), [
  { role: "analytics_read", db: "analytics" },
  { role: "read", db: "analytics" },
  { role: "sandbox_reader", db: "sandbox" }
]);

const customers = ensureCollection("appdb", "customers");
seed(customers, [
  { _id: 1, name: "Alice", status: "active", balance: NumberDecimal("120.50") },
  { _id: 2, name: "Bob", status: "paused", balance: NumberDecimal("40.00") },
  { _id: 3, name: "Carol", status: "active", balance: NumberDecimal("300.75") }
]);

const orders = ensureCollection("appdb", "orders");
seed(orders, [
  { _id: 1, customer_id: 1, total: NumberDecimal("42.00"), status: "paid" },
  { _id: 2, customer_id: 1, total: NumberDecimal("18.50"), status: "open" },
  { _id: 3, customer_id: 2, total: NumberDecimal("7.25"), status: "paid" }
]);

const audit = ensureCollection("appdb", "audit_log");
seed(audit, [
  { _id: 1, actor: "system", action: "bootstrap" },
  { _id: 2, actor: adminUser, action: "seed" }
]);

const events = ensureCollection("analytics", "events");
seed(events, [
  { _id: 1, event_type: "login", actor: "alice", payload: { source: "web" } },
  { _id: 2, event_type: "purchase", actor: "bob", payload: { order_id: 3 } },
  { _id: 3, event_type: "logout", actor: "alice", payload: {} }
]);

const notes = ensureCollection("sandbox", "notes");
seed(notes, [
  { _id: 1, body: "scratch database for live tests" },
  { _id: 2, body: "safe to reset" }
]);

print("mongo: fixture complete");
