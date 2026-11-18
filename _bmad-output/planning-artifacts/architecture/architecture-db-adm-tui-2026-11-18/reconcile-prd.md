# Reconciliation — PRD vs Architecture Spine

| PRD area | Spine landing | Status |
|---|---|---|
| Profiles, import, new connection (FR-31..36, FR-46) | AD-8, `profile` adapter, XDG schema | landed |
| Common tree, Security (FR-1..4) | AD-3, AD-4, `Catalog`, TUI tree | landed |
| Object details/stats (FR-5..9) | AD-3, AD-5, `StatsReader` | landed |
| Data browsing/search/paging (FR-10..18) | AD-5, AD-6, `DataReader`, `domain.Window` | landed |
| Query bar/results/errors (FR-19..26) | AD-6, AD-7, AD-10, `QueryExecutor` | landed |
| Read-only guard and mode (FR-27..30) | AD-7, AD-12, guard policy + session settings | landed |
| Privilege inspection (FR-37..41) | AD-3, AD-12, `PrivilegeReader` | landed |
| History (FR-42..45) | AD-9, `history` adapter | landed |
| NFR-1..5 performance/memory | AD-6, Verification performance checks | landed |
| NFR-6 error resilience | AD-11 | landed |
| NFR-7..9 secrets/ANSI/guard | AD-7, AD-8, AD-9, AD-11 | landed |
| NFR-10..14 no telemetry/static binary/terminal/versions/ops | AD-10, AD-13, Stack engine matrix, Deployment & Operations | landed |
| UJ-1..UJ-4 | Capability map + TUI flows (EXPERIENCE) | landed |

No product-level gaps remain. The only assumption is the module path, deferred to repo creation.
