# ADR-0014: Config schema versioning and migration runner

- **Status:** Accepted
- **Date:** 2026-06-28

## Context

Until now `config.Load` handled forward-compatibility purely through zero-value
backfill: a missing JSON key decodes to the type's zero value, and a few fields
(`Markers`, `ChannelFilters`, `Colors`) are re-seeded when empty. This is enough
to *add* fields but cannot express a transform — renaming a field, rewriting a
value's meaning, or splitting one setting into two — because by the time `Load`
sees the struct, `json.Unmarshal` has already discarded any old keys and there is
no record of which schema produced the file. `setup_wizard_version` exists, but it
governs whether the first-run wizard re-shows, not the shape of the config itself.
As the app accrues patches, some change will eventually need a real migration, and
retrofitting a version field onto already-deployed configs is harder the longer it
waits.

## Decision

**We will stamp every config with an explicit schema version and run versioned
migrations on load.** Config gains `config_version` (`int`); Go holds
`CurrentConfigVersion` (now `1`, the baseline schema in effect when versioning was
introduced). `Load` decodes a file's missing version as `0` (legacy), runs
`runConfigMigrations`, which applies each ordered step the config is behind, and
stamps `CurrentConfigVersion`. The migrated config is written back with the current
version on the next `Save`, so a legacy file is upgraded exactly once. `v0 → v1` is
deliberately a no-op: the baseline schema is unchanged, so existing zero-value
backfill already yields a valid `v1` config.

**Struct-based steps, with a documented escape hatch.** Each migration is a
`func(*Config)` operating on the already-decoded struct (KISS/YAGNI — that covers
value rewrites and field splits). A step that must read a *removed or renamed* JSON
key cannot see it on the struct; the ADR records that such a step would decode the
raw bytes into a map first. That seam is described, not built, until a real
migration needs it.

This intentionally extends the "No schema migrations" stance previously noted in
`docs/CODEMAPS/data.md`; that note is updated to point here.

## Consequences

- **Positive:** Future patches have a real upgrade path instead of being limited to
  additive, zero-value-compatible changes. Legacy configs migrate transparently and
  once. The version field is in place *before* it is first needed, which is the cheap
  time to add it.
- **Negative / risks:** A small amount of always-on machinery (one field, one runner)
  for a capability not yet exercised. The runner re-stamps the version on every load,
  so a config opened by an older build that does not understand a newer
  `config_version` would still load (Go ignores the unknown-higher version and only
  applies steps it has) — downgrades are not a supported path and are not guarded
  beyond "never apply a step you don't have."
- **Follow-up:** Add a migration step (and bump `CurrentConfigVersion`) when a future
  schema change cannot be expressed by zero-value backfill alone; add the raw-map
  decode seam at that point if the change renames or removes a key.
