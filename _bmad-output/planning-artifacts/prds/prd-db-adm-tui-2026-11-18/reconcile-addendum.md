# Reconciliation: addendum vs PRD

All five addendum sections were compared with the PRD. Since the PRD (§0) does not restate the addendum, each load-bearing detail must be an FR, NFR, glossary term, or explicit handoff. Wording-only differences are ignored. 15 gaps, 4 high-impact.

| Gap | Where in addendum | Where in PRD (or missing) | Destination | Impact |
|---|---|---|---|---|
| `Ctrl+X Ctrl+R` ("show native command") has no defined behavior | Keymap, line 39 | Missing; §7.2 defers "native command preview for grants" to v2 | PRD (decide v1 scope) | High: UX gets a binding the PRD neither requires nor defers |
| MySQL/MariaDB writes (`REPLACE`, `LOAD DATA`, `RENAME TABLE`, `SET GLOBAL`) absent from blocklist | "equivalent write/admin statements per dialect" | FR-28 list is PostgreSQL-centric (`COPY FROM`, `VACUUM`, `ANALYZE`) | PRD (rule) + architecture | High: `REPLACE INTO` bypasses the guard, breaking the v1 read-only promise |
| `~/.pgpass` must be `0600` or ignored — no read-side rule | Line 27 | NFR-7 sets `0600` only on files the tool creates | PRD (testable) | High: reads world-readable credential files, unlike psql |
| `~/.mongo.*` files are **offered for selection**, not merged | Line 29 | §7.1 "all `~/.mongo.*` files (including `~/.mongo`)" reads as import/merge; no picker | PRD + UX (picker) | High: may combine URIs from unrelated `~/.mongo.<user>` files |
| `~/.pgpass` matching: wildcards, first match wins, `\` escaping, `#` comments, `PGPASSFILE`/`passfile` override | Line 27 | Missing; "psql semantics" appears only in the brief | Architecture | Medium: wrong entry matched per host/port/database/user |
| `~/.mongo.*` precedence (last parsed line wins); malformed file → user fixes it | Line 29 | Missing | Architecture + PRD (error behavior) | Medium: nondeterministic URI choice; fallback hides bad credentials |
| Second read-only layer: PostgreSQL `default_transaction_read_only=on`, MySQL `SET SESSION TRANSACTION READ ONLY`, MongoDB none | Line 61 | §2.6 description only ("where supported"); no FR; Mongo limitation unrecorded | Architecture + one PRD line | Medium: layer dropped, or a story attempts an impossible Mongo mode |
| History path `~/.local/state/db-adm-tui/history` unstated | Line 66 | FR-24 "the history path" (undefined); NFR-14 "XDG paths" | Architecture | Medium: dangling term; PRD/brief diverge |
| Rotation: 10 MB × 5 files per format | Line 66 | §7.1 says only "rotation" | Architecture | Medium: unbounded or inconsistent growth |
| Redaction detail: literals written as `***`; passwords stay in memory for the session | Line 67 | NFR-7 says only "never appear"; no placeholder convention | Architecture | Low: may drop the statement or leak the literal |
| Audit fields: who, what, when, before/after, native command | Line 68 | §7.2 defers an "audit trail" with no content | PRD (§7.2 v2) | Low: v2 may omit before/after state already promised |
| `Ctrl+X i`: second key is plain `i`; `Ctrl+I` is `Tab` | Line 40 | FR-18 silent; FR-21 says only that Tab is never completion | UX | Low: chord may be specified as `Ctrl+X Ctrl+I`, indistinguishable from Tab |
| `Ctrl+X Ctrl+X` from "anywhere" vs "anywhere in the data workspace" | Line 36 | FR-20 narrows scope; "data workspace" undefined | UX | Low: unclear if it works from tree and details panes |
| Avoid `Ctrl+S` **and `Ctrl+Q`** | Line 54 | Only `Ctrl+S` noted (FR-9) | UX | Low: `Ctrl+Q` may later freeze the terminal |
| Landscape evidence, gap statement, sources; mongosh artifacts are not profiles | Lines 5–23, 29 | Not restated (§0) | Noise | None; only "no TUI imports `~/.pgpass`" could strengthen §1 |
