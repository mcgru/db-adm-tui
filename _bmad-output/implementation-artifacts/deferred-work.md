- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/1-foundation-and-tui-shell.md`
  summary: Below the 80x24 floor the picker is replaced by the too-small line, while EXPERIENCE.md allows the picker to keep rendering when it still fits.
  evidence: The story acceptance (SMALL_TERMINAL) requires only the too-small line, so the shell follows the story; the UX text and the story need a single ruling before a later story changes the behavior.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/1-foundation-and-tui-shell.md`
  summary: The tree width and query height are fixed in story 1; Ctrl+Left/Right/Up/Down are inert stubs and sizes do not persist per profile.
  evidence: The story fixes the 80x24 regions and has no persistence layer; EXPERIENCE.md -> Resize assigns the adjustable 16..half-width tree and 1..3-line query bar to the workspace polish story, so layout.Compute has no resize parameters or clamp branches yet.
