# Reconciliation — UX Spines vs Architecture Spine

| UX area | Spine landing | Status |
|---|---|---|
| Nightshift tokens and 17 components | `internal/tui/theme` maps DESIGN.md tokens to Lip Gloss; no literal hex outside theme | landed |
| 80×24 layout, top/bottom status, tree/main/query | TUI adapter, `domain.Grid`, `Layout` in TUI; terminal-too-small state | landed |
| Context tabs and focus model | TUI view state; app owns data; AD-4 | landed |
| Keymap and Latin/Cyrillic aliases | AD-10 central binding table + tests | landed |
| Completion and help | AD-10 local completion; `helpdata` + `make update-help` | landed |
| Read-only/read-write indicator and guard | AD-7, AD-12; app mode; engine session settings | landed |
| Results tab, errors, `results open` intent | AD-6 cursor lease + EXPERIENCE state row | landed |
| Table window vs query cursor | AD-6 | landed |
| Profile picker/new/import | AD-8, `profile` adapter, TUI modals | landed |
| Mockups | Linked from DESIGN/EXPERIENCE; not duplicated in spine | by reference |

No UX contract is contradicted. The query-result cursor lease is the only deliberate behavior addition; it is recorded in both spines.
