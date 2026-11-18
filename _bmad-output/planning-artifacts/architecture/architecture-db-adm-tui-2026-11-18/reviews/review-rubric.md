# Architecture Spine Review — db-adm-tui

## Overall verdict

Strong and lint-clean. The paradigm, dependency rule, stack, and deployment envelope are decided. Three seams were underspecified enough that two compliant units could diverge: the engine session bundle contract, the single-connection/cursor lease, and the SQL `database/sql` pool. They are fixed in the spine; the remaining medium findings are schema/contract precision.

## Findings

- **high — Session bundle contract.** AD-3 says "session bundle with optional ports plus Capabilities" but does not define the bundle shape, whether `Capabilities` is derived, or how nil ports and flags relate. *Fix:* pin `ports.SessionBundle` fields, derive capabilities from non-nil ports, and require a constructor test.
- **high — Query cursor lease.** AD-2 serializes all engine calls, while AD-6 keeps a forward-only query cursor open. The spine never says what happens to other engine-backed actions while Results streams. *Fix:* an open query cursor owns the session lease; the TUI shows `results open — Esc to release` and queues/rejects other engine operations.
- **high — `database/sql` pool.** `database/sql` may open more than one connection, contradicting AD-2. *Fix:* SQL adapters use a single `*sql.Conn` from a `sql.DB` configured with `MaxOpenConns=1`/`MaxIdleConns=1`, or bypass the pool.
- **medium — Profile schema.** AD-8 says "JSON profiles" without field names or paths. *Fix:* pin the v1 schema and XDG file path; require `0600` on the file.
- **medium — History schema.** AD-9 pins formats and rotation but not file names or record fields. *Fix:* pin `commands.jsonl` / `commands.log`, record fields, and the `***` placeholder.
- **medium — Keymap/help generation.** AD-10 says "central keymap" and `make update-help`, but not the binding table or help source layout. *Fix:* pin `keymap` as the sole binding table with Latin/Cyrillic aliases and tests; pin `helpdata/<engine>/*.md` plus a generated index validated against completion terms.
- **medium — Grid and order contracts.** `domain.Grid` lacks a `Kind` enum; table order is not exposed as a port. *Fix:* pin cell kinds and add `Catalog.OrderKey`.
