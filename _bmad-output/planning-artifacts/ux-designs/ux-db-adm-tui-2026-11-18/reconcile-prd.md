# PRD → UX Reconciliation

Reconciliation of `prd.md` (final, v1) against `DESIGN.md` and `EXPERIENCE.md`. Wording, token names, and purely visual choices are ignored. Impact: **blocker** = a v1 requirement is unreachable or contradicted; **should-fix** = a testable behavior is dropped, narrowed, or added without PRD support; **note** = lower-risk traceability gap.

| Gap | PRD location | Spine location | Impact |
|---|---|---|---|
| Details and Data are tabs in one main pane; a table or collection opens on Data only. PRD requires an adjacent details pane and data view opened together, with `Tab` moving focus tree → details → data. | §2.1 FR-4; §2.2 intro | EXPERIENCE Information Architecture “Context tabs”; Interaction Primitives “Focus”; tree/data-grid patterns | blocker |
| Views and sequences get `Details` + `Privileges` tabs and open on Details. PRD makes them tree-only nodes, requires the `detail view not available in v1` placeholder, and explicitly schedules no detail/data panes. | §2.1 FR-2; §6 Non-Goals; §7.2 | EXPERIENCE Context tabs; tree pattern; DESIGN object-tabs | blocker |
| No surface or navigation entry exposes the connected user’s own privileges; `Privileges` exists only on database/schema/table/role selections. FR-37 is unreachable. | §2.8 FR-37; §2.1 FR-3 | EXPERIENCE Information Architecture tab table; privileges-pane | blocker |
| `~/.pgpass` lines are individually selectable with `Space`; PRD auto-imports every valid line. The `0600` permission rejection and `PGPASSFILE` override are also absent. | §2.7 FR-32, FR-31 | EXPERIENCE import-modal; State Patterns | should-fix |
| A malformed selected `~/.mongo.*` file is shown as a warning row while valid lines import; PRD requires an error naming the file and no import from it. | §2.7 FR-33 (contrast FR-32) | EXPERIENCE import-modal; “Malformed import line” | should-fix |
| Query bar is adjustable to 1–8 lines and persisted per profile; PRD defines a one-line query bar, with multi-line editing confined to the edit modal. | §2.5 intro, FR-19; FR-23 | DESIGN Layout & Spacing, query-bar; EXPERIENCE `Ctrl+Up/Down` | should-fix |
| Privilege view omits MySQL granted-vs-active roles, PostgreSQL role attributes/native scope labels, MongoDB inherited roles, and the “effective unavailable → native grants + explicit note” state. | §2.1 FR-3; §2.8 FR-37, FR-39 | EXPERIENCE privileges-pane; Flow F2 | should-fix |
| MongoDB Security is one connection-root node holding users and roles; PRD requires users/roles per database and cluster roles reachable from the admin database. | §2.1 FR-3 | EXPERIENCE tree; Information Architecture | should-fix |
| History redaction is absent: no rule for password literals or connection URIs in stored/recalled statements and error or crash text. Corrupted-file warning is also missing; only “not writable” is stated. | §2.9 FR-43, FR-45 | EXPERIENCE query-bar; State Patterns “History failure” | should-fix |
| End-of-data gesture omits its disable while the filter prompt is open, counter reset on any other key or focus move, and the 10,000 active batch after `Ctrl+A`. | §2.3 FR-12, FR-13 | EXPERIENCE data-grid, filter-bar | note |
| UX-added scope: per-object `ro`/`rw` tree markers, `Privileges` tabs on database/schema, `?` keymap, `q` quit, and adjustable panes have no PRD requirement. | §2.1 FR-3; §6; §7.2 | DESIGN tree, object-tabs; EXPERIENCE IA, keymap | note |
| Cyrillic physical-key guarantee is stated only for `n`/`i`/`q`/`m`; chord letters `s`, `e`, `w` and `N` are not covered. | NFR-12 | EXPERIENCE Accessibility Floor | note |
