// Package api is the Wails binding layer. The App struct's exported methods are
// what the frontend calls; it wires together config, parsing, highlighting,
// search, tags, reassembly, and the filesystem watcher. All log access is
// read-only.
package api

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gobchat-log-browser/internal/config"
	"gobchat-log-browser/internal/highlight"
	"gobchat-log-browser/internal/i18n"
	"gobchat-log-browser/internal/logstore"
	"gobchat-log-browser/internal/parser"
	"gobchat-log-browser/internal/reassemble"
	"gobchat-log-browser/internal/search"
	"gobchat-log-browser/internal/tags"
	"gobchat-log-browser/internal/update"
	"gobchat-log-browser/internal/version"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the bound application object.
type App struct {
	ctx context.Context

	mu         sync.RWMutex
	cfg        config.Config
	configPath string

	index *search.Index
	store *logstore.LogStore
	tags  *tags.TagStore
	loc   *i18n.Localizer

	// watcherMu serializes startWatcher's close-old/create-new swap; a.mu still
	// guards the watcher field itself (Shutdown only closes, so it needs a.mu only).
	watcherMu sync.Mutex
	watcher   *logstore.Watcher

	// debounce coalesces the burst of Write events fsnotify reports while a log
	// is actively being appended to, so each burst causes one re-parse.
	debounceMu sync.Mutex
	debounce   map[string]*time.Timer
}

// fileChangeDebounce is how long a file must stay quiet after a Write event
// before it is re-parsed and the frontend is notified.
const fileChangeDebounce = 300 * time.Millisecond

// NewApp allocates the App. Heavy initialisation happens in Startup, once Wails
// provides the context.
func NewApp() *App {
	return &App{debounce: map[string]*time.Timer{}}
}

// GetVersion returns the app version ("dev" for local builds).
func (a *App) GetVersion() string {
	return version.Version
}

// CheckForUpdate queries GitHub for the latest release and compares it against
// the running version (docs/adr/0012). Dev builds skip the network call
// entirely. Callers decide how to surface errors: the startup check swallows
// them (offline must be silent), the manual Settings check displays them.
func (a *App) CheckForUpdate() (UpdateCheckResult, error) {
	res := UpdateCheckResult{CurrentVersion: version.Version}
	if _, ok := update.ParseVersion(version.Version); !ok {
		res.Status = "dev"
		return res, nil
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	rel, err := update.NewClient().LatestRelease(ctx)
	if err != nil {
		return res, fmt.Errorf("%s: %w", a.t("error.updateCheckFailed"), err)
	}
	if _, ok := update.ParseVersion(rel.TagName); !ok {
		return res, fmt.Errorf("%s: unexpected tag %q", a.t("error.updateCheckFailed"), rel.TagName)
	}

	res.LatestVersion = strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
	res.ReleaseURL = rel.HTMLURL
	if update.IsNewer(rel.TagName, version.Version) {
		res.Status = "update_available"
	} else {
		res.Status = "up_to_date"
	}
	return res, nil
}

// Startup is wired to Wails OnStartup. It loads config and tags, builds the
// search index and log store, performs an initial scan, and starts watching.
// Load failures fall back to safe defaults but are reported in a warning
// dialog rather than swallowed.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	cfgPath, _ := config.ConfigFilePath()
	a.configPath = cfgPath
	cfg, cfgErr := config.Load(cfgPath)

	tagsPath, _ := config.TagsFilePath()
	tagStore, tagsErr := tags.NewTagStore(tagsPath)
	corruptTagsPath := tagsPath + ".corrupt"
	if tagsErr != nil {
		// Preserve the unreadable sidecar instead of silently forking to a
		// different file, then start fresh at the canonical path so future
		// saves land where the next load looks.
		if os.Rename(tagsPath, corruptTagsPath) == nil {
			tagStore, _ = tags.NewTagStore(tagsPath)
		}
		if tagStore == nil {
			tagStore = tags.NewEmptyTagStore(tagsPath)
		}
	}

	loc, _ := i18n.New(cfg.Language)

	a.mu.Lock()
	a.cfg = cfg
	a.tags = tagStore
	a.loc = loc
	a.index = search.NewIndex()
	// The metadata cache is derived data: if index.json is missing or corrupt,
	// LoadMetaCache returns an empty cache and the scan simply parses everything.
	idxPath, _ := config.IndexFilePath()
	a.store = logstore.New(a.index, logstore.LoadMetaCache(idxPath))
	a.mu.Unlock()

	var warnings []string
	if cfgErr != nil {
		warnings = append(warnings, a.t("warn.configLoadFailed"))
	}
	if tagsErr != nil {
		warnings = append(warnings, fmt.Sprintf(a.t("warn.tagsLoadFailed"), corruptTagsPath))
	}
	if len(warnings) > 0 {
		// Off the startup path so the window appears first; runtime dialogs are
		// safe to call from goroutines.
		go func() {
			_, _ = wruntime.MessageDialog(a.ctx, wruntime.MessageDialogOptions{
				Type:    wruntime.WarningDialog,
				Title:   a.t("warn.startupTitle"),
				Message: strings.Join(warnings, "\n"),
			})
		}()
	}

	// The initial scan can touch hundreds of files; run it off the startup path
	// so the window appears immediately, then tell the frontend to refresh.
	go func() {
		_ = a.store.ScanAll(a.effectiveDirs())
		a.startWatcher()
		if a.ctx != nil {
			wruntime.EventsEmit(a.ctx, "logs:scanned")
		}
	}()
}

// Shutdown is wired to Wails OnShutdown to release the watcher.
func (a *App) Shutdown(context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.watcher != nil {
		_ = a.watcher.Close()
		a.watcher = nil
	}
}

// --- Config ---

// GetConfig returns the current configuration.
func (a *App) GetConfig() config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

// SaveConfig persists new settings and applies side effects: reloads the
// localizer if the language changed and rescans/rewatches if directories or
// auto-detect changed.
func (a *App) SaveConfig(cfg config.Config) error {
	a.mu.Lock()
	old := a.cfg
	a.cfg = cfg
	path := a.configPath
	if cfg.Language != old.Language {
		if loc, err := i18n.New(cfg.Language); err == nil {
			a.loc = loc
		}
	}
	a.mu.Unlock()

	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("%s: %w", a.t("error.configSaveFailed"), err)
	}

	dirsChanged := !equalStrings(old.LogDirectories, cfg.LogDirectories) ||
		old.AutoDetectAppData != cfg.AutoDetectAppData
	if dirsChanged {
		_ = a.store.ScanAll(a.effectiveDirs())
		a.startWatcher()
	}
	return nil
}

// --- Log discovery ---

// ScanLogs rescans all effective directories and returns the updated summaries.
func (a *App) ScanLogs() ([]LogSummary, error) {
	if err := a.store.ScanAll(a.effectiveDirs()); err != nil {
		return nil, fmt.Errorf("%s: %w", a.t("error.scanFailed"), err)
	}
	return a.GetLogList(), nil
}

// GetLogList returns the cached summaries without rescanning.
func (a *App) GetLogList() []LogSummary {
	metas := a.store.List()
	out := make([]LogSummary, 0, len(metas))
	for _, m := range metas {
		out = append(out, a.summary(m))
	}
	return out
}

// --- Log viewing ---

// GetLogEntries parses (read-only) and returns all entries for a file with
// highlight spans pre-computed.
func (a *App) GetLogEntries(filePath string) ([]EntryDTO, error) {
	entries, err := a.store.GetEntries(filePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", a.t("error.parseFailed"), err)
	}
	a.mu.RLock()
	markers := a.cfg.Markers
	mentions := a.cfg.MentionNames
	a.mu.RUnlock()

	out := make([]EntryDTO, len(entries))
	for i, e := range entries {
		out[i] = toEntryDTO(e, markers, mentions)
	}
	return out, nil
}

// GetLogThreads returns the optional reassembled (in-memory) view of a file.
func (a *App) GetLogThreads(filePath string) ([]ThreadDTO, error) {
	entries, err := a.store.GetEntries(filePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", a.t("error.parseFailed"), err)
	}
	a.mu.RLock()
	markers := a.cfg.Markers
	mentions := a.cfg.MentionNames
	a.mu.RUnlock()

	threads := reassemble.Reassemble(entries)
	out := make([]ThreadDTO, len(threads))
	for i, th := range threads {
		out[i] = ThreadDTO{
			Sender:    th.Sender,
			Channel:   string(th.Channel),
			Lines:     th.Lines,
			Combined:  th.Combined,
			Spans:     highlight.Tokenize(th.Combined, markers, mentions),
			StartTime: isoTime(th.StartTime),
			EndTime:   isoTime(th.EndTime),
		}
	}
	return out, nil
}

// --- Search ---

// Search runs a query. filePath empty means global search across all known
// logs; otherwise the search is restricted to that file. channels and sender
// are optional post-filters.
func (a *App) Search(text, filePath string, channels []string, sender string) ([]SearchResultDTO, error) {
	// Ensure the relevant files are parsed and indexed.
	if filePath != "" {
		if _, err := a.store.GetEntries(filePath); err != nil {
			return nil, fmt.Errorf("%s: %w", a.t("error.parseFailed"), err)
		}
	} else {
		for _, m := range a.store.List() {
			_, _ = a.store.GetEntries(m.FilePath)
		}
	}

	results := a.index.Query(text, filePath, 0)
	channelSet := toSet(channels)
	senderLower := strings.ToLower(strings.TrimSpace(sender))

	out := make([]SearchResultDTO, 0, len(results))
	for _, r := range results {
		entry, ok := a.entryAt(r.FilePath, r.LineNumber)
		if !ok {
			continue
		}
		if len(channelSet) > 0 {
			if _, want := channelSet[string(entry.Channel)]; !want {
				continue
			}
		}
		if senderLower != "" && !strings.Contains(strings.ToLower(entry.DisplayName), senderLower) {
			continue
		}
		out = append(out, SearchResultDTO{
			FilePath:   r.FilePath,
			FileName:   baseName(r.FilePath),
			LineNumber: r.LineNumber,
			Channel:    string(entry.Channel),
			Sender:     entry.DisplayName,
			Snippet:    entry.Message,
			Score:      r.Score,
		})
	}
	return out, nil
}

// --- Tags ---

func (a *App) GetTags(fileName string) tags.FileTags { return a.tags.GetTags(fileName) }

func (a *App) SetTags(fileName string, tagList []string, note string) error {
	return a.tags.SetTags(fileName, tagList, note)
}

func (a *App) GetAllTagNames() []string { return a.tags.AllTags() }

// --- First-run setup ---

// GetSetupState reports whether the first-run wizard should be shown. It is
// needed when no config file exists yet, when there is no usable log directory
// (the detected default does not exist and no configured directory exists), or
// when the wizard gained new content since the user last completed it. It also
// returns the detected default directory to prefill the wizard and consumes
// the installer's one-shot update-check seed, if present.
func (a *App) GetSetupState() SetupState {
	a.mu.RLock()
	cfgPath := a.configPath
	cfg := a.cfg
	a.mu.RUnlock()

	st := SetupState{}
	if cfgPath != "" {
		if _, err := os.Stat(cfgPath); err == nil {
			st.ConfigExists = true
		}
	}
	if def, err := config.GobchatDefaultLogDir(); err == nil {
		st.DefaultLogDir = def
		if fi, statErr := os.Stat(def); statErr == nil && fi.IsDir() {
			st.DefaultLogDirExists = true
		}
	}

	anyConfigured := false
	for _, d := range cfg.LogDirectories {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			anyConfigured = true
			break
		}
	}

	st.WizardVersion = config.SetupWizardCurrentVersion
	if seedPath, err := config.InstallerDefaultsFilePath(); err == nil {
		if def, found := config.ConsumeInstallerDefaults(seedPath); found {
			st.InstallerSeedFound = true
			st.InstallerCheckUpdates = def.CheckUpdatesOnStart
		}
	}

	st.NeedsSetup = needsSetup(st.ConfigExists, st.DefaultLogDirExists, anyConfigured, cfg.SetupWizardVersion)
	return st
}

// needsSetup decides whether the wizard shows: on true first run, when no
// usable log directory exists, or when the wizard gained new content since the
// user last completed it (savedWizardVersion behind the current version).
func needsSetup(configExists, defaultDirExists, anyConfigured bool, savedWizardVersion int) bool {
	return !configExists ||
		(!defaultDirExists && !anyConfigured) ||
		savedWizardVersion < config.SetupWizardCurrentVersion
}

// --- Dialogs ---

// PickDirectory opens the native directory picker and returns the chosen path
// (empty string if the user cancels).
func (a *App) PickDirectory() (string, error) {
	if a.ctx == nil {
		return "", nil
	}
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Select a Gobchat log directory",
	})
}

// --- i18n ---

// GetLocaleMessages returns the backend's localized strings for the active
// language, for the frontend to merge into its own translations.
func (a *App) GetLocaleMessages() map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.loc == nil {
		return map[string]string{}
	}
	return a.loc.Messages()
}

// --- internals ---

func (a *App) t(key string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.loc == nil {
		return key
	}
	return a.loc.T(key)
}

// effectiveDirs returns the directories to scan: the configured ones plus, when
// auto-detect is on, Gobchat's default log directory (detected at runtime, not
// stored in config).
func (a *App) effectiveDirs() []string {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()

	seen := map[string]struct{}{}
	var dirs []string
	add := func(d string) {
		if d == "" {
			return
		}
		if _, ok := seen[d]; ok {
			return
		}
		seen[d] = struct{}{}
		dirs = append(dirs, d)
	}
	if cfg.AutoDetectAppData {
		if d, err := config.GobchatDefaultLogDir(); err == nil {
			add(d)
		}
	}
	for _, d := range cfg.LogDirectories {
		add(d)
	}
	return dirs
}

func (a *App) startWatcher() {
	// Serialize close-old/create-new: two concurrent calls (e.g. SaveConfig
	// racing ScanLogs) would otherwise each create a watcher and leak one.
	a.watcherMu.Lock()
	defer a.watcherMu.Unlock()

	a.mu.Lock()
	if a.watcher != nil {
		_ = a.watcher.Close()
		a.watcher = nil
	}
	a.mu.Unlock()

	// Watch every directory the scan discovered (roots plus their subfolders),
	// not just the configured roots, since fsnotify is not recursive.
	dirs := a.store.WatchDirs()
	if len(dirs) == 0 {
		dirs = a.effectiveDirs()
	}
	w, err := logstore.Watch(dirs, a.onFileChange)
	if err != nil {
		return
	}
	a.mu.Lock()
	a.watcher = w
	a.mu.Unlock()
}

func (a *App) onFileChange(path string, ev logstore.WatchEvent) {
	a.debounceMu.Lock()
	if t, ok := a.debounce[path]; ok {
		t.Stop()
		delete(a.debounce, path)
	}
	if ev == logstore.WatchEventModified {
		a.debounce[path] = time.AfterFunc(fileChangeDebounce, func() {
			a.debounceMu.Lock()
			delete(a.debounce, path)
			a.debounceMu.Unlock()
			a.handleFileChange(path, ev)
		})
		a.debounceMu.Unlock()
		return
	}
	a.debounceMu.Unlock()
	a.handleFileChange(path, ev)
}

func (a *App) handleFileChange(path string, ev logstore.WatchEvent) {
	_ = a.store.Refresh(path)
	if a.ctx == nil {
		return
	}
	switch ev {
	case logstore.WatchEventCreated:
		if m, ok := a.store.Get(path); ok {
			wruntime.EventsEmit(a.ctx, "log:new", a.summary(m))
		}
	case logstore.WatchEventModified:
		wruntime.EventsEmit(a.ctx, "log:updated", path)
	case logstore.WatchEventRemoved:
		wruntime.EventsEmit(a.ctx, "log:removed", path)
	}
}

func (a *App) entryAt(filePath string, line int) (parser.LogEntry, bool) {
	entries, err := a.store.GetEntries(filePath)
	if err != nil {
		return parser.LogEntry{}, false
	}
	// Line numbers are 1-based and dense, but tolerate gaps by scanning.
	if line-1 >= 0 && line-1 < len(entries) && entries[line-1].LineNumber == line {
		return entries[line-1], true
	}
	for _, e := range entries {
		if e.LineNumber == line {
			return e, true
		}
	}
	return parser.LogEntry{}, false
}

func (a *App) summary(m logstore.LogMeta) LogSummary {
	channels := make([]string, len(m.Channels))
	for i, c := range m.Channels {
		channels[i] = string(c)
	}
	ft := a.tags.GetTags(m.FileName)
	return LogSummary{
		FilePath:     m.FilePath,
		FileName:     m.FileName,
		LogDate:      isoTime(m.LogDate),
		MessageCount: m.MessageCount,
		Participants: m.Participants,
		Channels:     channels,
		Duration:     formatDuration(m.Duration),
		Tags:         ft.Tags,
		Note:         ft.Note,
	}
}

func toEntryDTO(e parser.LogEntry, markers highlight.MarkerSet, mentions []string) EntryDTO {
	return EntryDTO{
		LineNumber:      e.LineNumber,
		Channel:         string(e.Channel),
		Timestamp:       isoTime(e.Timestamp),
		Sender:          e.Sender,
		DisplayName:     e.DisplayName,
		Realm:           e.Realm,
		StatusSymbol:    e.StatusSymbol,
		Message:         e.Message,
		Spans:           highlight.Tokenize(e.Message, markers, mentions),
		PartIndex:       e.PartIndex,
		PartTotal:       e.PartTotal,
		IsContinuation:  e.IsContinuation,
		HasContinuation: e.HasContinuation,
	}
}

func isoTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	d = d.Round(time.Minute)
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

func baseName(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[i+1:]
	}
	return path
}

func toSet(items []string) map[string]struct{} {
	if len(items) == 0 {
		return nil
	}
	s := make(map[string]struct{}, len(items))
	for _, it := range items {
		s[it] = struct{}{}
	}
	return s
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
