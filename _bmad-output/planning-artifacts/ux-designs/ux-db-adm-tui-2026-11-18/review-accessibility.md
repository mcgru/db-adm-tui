# Accessibility Review — db-adm-tui UX Spines

**Verdict: conditional pass.** The color system is the strongest part of the pair: all ten stated contrast pairs reproduce exactly (18.61, 7.70, 11.27, 14.06, 10.25, 9.32, 7.01, 6.39, 12.02, 12.34), and every state carries a word or glyph. What fails is the behavior layer — horizontal access, long-pane scrolling, and status-line arithmetic at the 80×24 floor. Fix the two critical items before implementation.

## Critical

**C1. No key path for horizontal movement in data/results.**
`DESIGN.md:341` and `EXPERIENCE.md:228` promise horizontal scrolling; `EXPERIENCE.md:279` uses a "cell cursor"; `Ctrl+F` acts on "the focused cell" (`EXPERIENCE.md:181`). But `Left`/`Right` are scoped to the tree only (`EXPERIENCE.md:172`) and no Home/End or column keys exist (`EXPERIENCE.md:166-197`). The grid has ~47 inner cells at the floor (`DESIGN.md:305`), so most columns of a wide table are unreachable.
*Fix:* add `Left`/`Right`, `Home`/`End` (or `Ctrl+Left/Right` for column edge) in data/results; name the cell cursor as the focused field.

**C2. Details and privileges panes have no scrolling.**
`Up`/`Down` scope is "tree, data, results, help, query bar" (`EXPERIENCE.md:171`); details (`EXPERIENCE.md:97`) and privileges (`EXPERIENCE.md:98`) are absent. The main pane holds ~18 content rows at 24 lines (`DESIGN.md:305`). A 30-column structure or a role with many grants is cut off with no way to read it — the core of UJ-1.
*Fix:* scroll keys when the pane is focused, with a position indicator.

## High

**H1. Status line cannot hold message plus hints at 80 columns.**
`Blocked: UPDATE is a write statement. Read-only mode is on. Press Ctrl+X Ctrl+W to switch.` is 90 characters (`EXPERIENCE.md:149`). With right-aligned hints (`DESIGN.md:336`) the remediation is what gets cut, and no truncation rule covers messages — only counts (`EXPERIENCE.md:228`).
*Fix:* state priority explicitly: hint collapses first; message truncates at a word boundary, preserving the key sentence.

**H2. Import modal and profile picker are unbounded vertically.**
Modals are "content + two border lines" with 2–3 cell margins (`DESIGN.md:311`); the import modal lists every discovered line with no viewport (`DESIGN.md:348`); the picker renders rows "if it fits" (`EXPERIENCE.md:225`). A 40-line `~/.pgpass` overflows 24 lines.
*Fix:* internal scroll viewport plus count line and keys for both lists.

**H3. Modal keyboard model is undefined.**
`Tab` cycles three workspace zones (`EXPERIENCE.md:43`); no modal trap, field order, or intra-modal focus cue exists, yet the new-connection modal has two inputs (`DESIGN.md:347`) and the workspace behind is inactive (`DESIGN.md:319`).
*Fix:* modals trap `Tab`, define field order, and show a focused-field marker (leading `>` plus focus border).

**H4. Results error wrapping contradicts floor arithmetic.**
The engine message wraps at 72 cells (`DESIGN.md:345`) inside a 47-cell main pane (`DESIGN.md:305`).
*Fix:* wrap at pane inner width; keep 72 as a modal-only measure.

**H5. Access markers are dimmed below the contrast floor.**
`ro`/`rw`/`—` use `text-secondary` plus the dim attribute (`DESIGN.md:337`). At 50% dim, `#94A6B8` on `#0A0F15` is ≈2.7:1, contradicting "all text meets at least 4.5:1" (`EXPERIENCE.md:209`). Dim is terminal-dependent, so no listed pair can back that claim.
*Fix:* markers at full `text-secondary`; reserve dim for non-essential text.

## Medium

**M1. Query-bar height is specified as 1–8 lines** (`DESIGN.md:301`) **and 1–3** (`DESIGN.md:307`, `EXPERIENCE.md:194`). Pick 3 and correct the diagram.

**M2. Completion popup shows at most 8 suggestions with no beyond-8 navigation** (`DESIGN.md:344`, `EXPERIENCE.md:102`). Add `PgUp`/`PgDn` or a scroll indicator.

**M3. Chord finals are documented two ways:** `Ctrl+X w`/`e` (`EXPERIENCE.md:212`) versus `Ctrl+X Ctrl+W`/`Ctrl+X Ctrl+E` (`EXPERIENCE.md:185`, `prd.md:382`). The 2-second timeout is invisible and an unknown final key cancels silently (`EXPERIENCE.md:164`); add a `cancelled` status.

**M4. "Every status color is paired with a word"** (`DESIGN.md:260`) **is not operationalized.** Tree, details, and privileges errors specify color only (`DESIGN.md:98`, `:113`, `:125`). Require a leading `error`/`warning` tag.

**M5. The new-connection `uri` field is plain text with no rule for embedded credentials** (`DESIGN.md:347`, `EXPERIENCE.md:105`), conflicting with the redacted-string requirement (`prd.md:457`). Reject or redact `user:pass@` on entry and in error output.

**M6. Sanitization is scoped to "database content"** (`EXPERIENCE.md:213`, `prd.md:659`), but pasted query text, `.mongo`/`.pgpass`-derived values, and source filenames are also rendered. Sanitize all external strings.

## Low

**L1.** Truncation priority for `privileged` and the database name at 80 columns is unstated (`EXPERIENCE.md:93`, `:228`); keep the `privileged` tag.

**L2.** The confirm modal says "the default is cancel" but `Enter` confirms (`EXPERIENCE.md:107`); with no pointer a "default" is meaningless. Consider a typed word only for read-write.

**L3.** The terminal-too-small transition is undefined when a modal is open or a query is running (`EXPERIENCE.md:157`, `:225`).

**Screen reader (explicitly out of scope, `EXPERIENCE.md:211`) — residual risks only:** no live-region analogue for the status lines, once-per-second elapsed updates (`EXPERIENCE.md:138`), five-second transient messages (`:94`), and cursor-addressed modals read as unstructured lines. Not defects.

## Resolution (post-review)

All critical and high findings were applied to the spines: horizontal cell movement and Home/End; details and privileges scrolling; bounded picker/import viewports with modal focus trap; shortened blocked-statement copy with key-hint priority; results error wrapping at pane width; access markers at full `text-secondary`; query bar fixed at 1–3 lines; one chord notation; URI credentials rejected; sanitization extended to all external strings. The two critical items are closed.
