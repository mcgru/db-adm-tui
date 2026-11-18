# Adversarial Review — db-adm-tui

Two units built independently, each obeying every AD literally, could still build incompatibly in these pairs:

1. **Adapter A returns `PrivilegeReader` non-nil with `Capabilities.Privileges=false`; Adapter B returns nil with `Capabilities.Privileges=true`.** The app branches differently on the same engine. **Fix:** `ports.SessionBundle` is the single source; capabilities are derived from non-nil ports; a constructor test enforces the mapping.

2. **SQL adapter uses `sql.DB` directly; Mongo adapter uses one client.** One honors AD-2's "one connection", the other does not. **Fix:** SQL adapters use one `*sql.Conn` from `sql.DB` with `MaxOpenConns=1`/`MaxIdleConns=1`; AD-2 names this.

3. **A query result cursor stays open while the user switches to Details.** One implementation queues the stats request; another opens a second connection. **Fix:** an open cursor owns the session lease; other engine operations queue/reject with `busy`; the TUI exposes `results open — Esc to release`.

4. **Adapter A orders tables by primary key; Adapter B relies on natural order.** Same UI, different rows on re-fetch. **Fix:** `Catalog.OrderKey(ctx, ObjectRef)` is a port; SQL uses the primary key or warns; Mongo uses `_id`.

5. **Adapter A redacts with `***`; history B redacts with `[redacted]`.** History files diverge. **Fix:** AD-9 pins the placeholder and fields.

6. **Keymap A normalizes Cyrillic `Ctrl+W`; keymap B only accepts Latin `Ctrl+W`.** Same chord, different behavior. **Fix:** AD-10 pins one binding table and requires tests for every letter/chord final in both layouts.

7. **Help generator A indexes only SQL keywords; generator B indexes all driver terms.** Completion/help drift. **Fix:** `make update-help` validates every completion term across enabled drivers and fails on a missing entry.

All seven are closed in the spine.
