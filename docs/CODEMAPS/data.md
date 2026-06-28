<!-- Generated: 2026-06-19 | Files scanned: 34 Go | Token estimate: ~700 -->

# Data Model

No database. Persistence is JSON files written atomically; source logs are
strictly read-only (ADR-0007). App data lives in `%APPDATA%\GobchatLogBrowser`
(separate from Gobchat's own `%APPDATA%\Gobchat\log`).

## Persistent stores (on disk)

```
config.json          config.Config         user settings (config/config.go)
<tags file>          tags.TagStore         filename-keyed { tags[], note } (ADR-0005)
index.json           logstore.MetaCache    metadata cache for fast startup (ADR-0009)
<installer seed>     installer defaults     one-shot update-check opt-in (ADR-0012)
webview2/            WebView2 user data     kept in app dir, not %APPDATA%\<exe> (ADR-0010)
```

Paths resolved in `config/paths.go` (AppDataDir, ConfigFilePath, TagsFilePath,
IndexFilePath, InstallerDefaultsFilePath, GobchatDefaultLogDir).

## Schemas

```
Config (config.json)
  log_directories[]        configured roots
  auto_detect_appdata      also scan Gobchat default dir (runtime-detected)
  language                 "en" | "de"
  mention_names[]          highlighted as mentions
  roleplay_characters[]    pinned in player filter
  markers                  MarkerSet{speech[],emote[],ooc[]} (open/close pairs)
  theme                    "light" | "dark"
  channel_filters          map[channel]bool
  check_updates_on_start   opt-in (default false — never phone home w/o consent)
  setup_wizard_version     last completed wizard version (0 = pre-versioning)
  colors                   theme → category → hex override

FileTags (tags store, keyed by log file name)
  tags[]    note

LogMeta (index.json cache + in-memory registry; logstore/scanner.go)
  file_path file_name folder log_date message_count participants[]
  channels[] first_entry last_entry duration size_bytes
```

## In-memory (not persisted)

```
parser.ParsedLog   {FilePath, Version, FormatStr, Entries[], ParseErrors[]}  lazy, cached per file
search.Index       map[token][]Posting{FilePath,LineNumber} — rebuilt each run
reassemble.Thread  sender-keyed in-memory join of entries (ADR-0007); never on disk
```

## Source log format (read-only input)

```
Line 1: Chatlogger Id: FCLv1 | CCLv1
Line 2 (CCLv1): Chatlogger format:{channel} [{date} {time-full}] {sender}: {message}
e.g. Say [2026-05-16 20:09:30+02:00] ★Max Mustermiqote [Shiva]: "Hello..." (1/2)
Filenames: chatlog_YYYY-MM-DD_HH-mm.log
```

Unparseable lines surface as `ChannelUnknown` with raw text — never dropped.

## "Migrations"

No schema migrations. Forward-compat is handled at load time:
- `config.Load` backfills empty marker sets with `DefaultMarkerSet`; missing
  fields decode to zero values; missing file → defaults (not an error).
- `setup_wizard_version` < `SetupWizardCurrentVersion` (=2) re-shows the wizard
  with new content (history: 1=original, 2=update-check opt-in).
- Corrupt tags file is renamed `.corrupt` and a fresh store starts at the
  canonical path (Startup); corrupt/missing `index.json` → empty cache, re-parse.
