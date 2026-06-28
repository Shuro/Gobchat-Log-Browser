// Package velopackupd wraps the Velopack update manager (via velopack-go) behind
// a small, GUI-agnostic API: check, then download+apply. It is only meaningful
// for an installed Velopack build; dev and otherwise-unsupported builds report
// "not installed" so callers can stay silent. Replaces the old GitHub-API
// notify-only checker (docs/adr/0013).
package velopackupd

import (
	"fmt"
	"sync"

	"github.com/quaadgras/velopack-go/velopack"
)

// Status mirrors the subset of update outcomes the frontend cares about.
type Status string

const (
	// StatusNotInstalled means this build can't self-update (dev build, portable,
	// or velopack-go reports the app isn't a Velopack install). The UI hides the
	// updater entirely for this state.
	StatusNotInstalled Status = "dev"
	StatusUpToDate     Status = "up_to_date"
	StatusAvailable    Status = "update_available"
)

// Result is the outcome of a Check.
type Result struct {
	Status  Status
	Current string
	Latest  string
}

// Updater holds the Velopack manager and the most recent pending update so a
// later DownloadAndApply doesn't have to re-check. It is safe for concurrent use.
type Updater struct {
	feedURL string

	mu      sync.Mutex
	mgr     *velopack.UpdateManager
	pending *velopack.UpdateInfo
}

// New returns an Updater that reads releases from feedURL (a Velopack HTTP feed,
// e.g. a GitHub "releases/latest/download" base URL).
func New(feedURL string) *Updater {
	return &Updater{feedURL: feedURL}
}

// manager lazily constructs the Velopack manager. Construction fails when the
// running app is not a Velopack install (dev/portable), which we surface as
// "not installed" rather than an error.
func (u *Updater) manager() (*velopack.UpdateManager, bool) {
	if u.mgr != nil {
		return u.mgr, true
	}
	m, err := velopack.NewUpdateManager(u.feedURL)
	if err != nil {
		return nil, false
	}
	u.mgr = m
	return m, true
}

// Check queries the feed for a newer release. A build that can't self-update
// returns StatusNotInstalled with no error.
func (u *Updater) Check() (Result, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	m, ok := u.manager()
	if !ok {
		return Result{Status: StatusNotInstalled}, nil
	}
	current := m.CurrentlyInstalledVersion()
	info, status, err := m.CheckForUpdates()
	if err != nil {
		return Result{Status: StatusNotInstalled, Current: current}, fmt.Errorf("velopack check failed: %w", err)
	}
	if status != velopack.UpdateAvailable || info == nil || info.TargetFullRelease == nil {
		return Result{Status: StatusUpToDate, Current: current, Latest: current}, nil
	}
	u.pending = info
	return Result{Status: StatusAvailable, Current: current, Latest: info.TargetFullRelease.Version}, nil
}

// DownloadAndApply downloads the pending update (re-checking if needed),
// reporting 0–100 progress, then asks the Velopack updater to wait for this
// process to exit and apply+restart. The caller MUST quit the app once this
// returns nil so the updater can take over.
func (u *Updater) DownloadAndApply(progress func(percent uint)) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	m, ok := u.manager()
	if !ok {
		return fmt.Errorf("velopack: this build cannot self-update")
	}
	info := u.pending
	if info == nil {
		fresh, status, err := m.CheckForUpdates()
		if err != nil {
			return fmt.Errorf("velopack check failed: %w", err)
		}
		if status != velopack.UpdateAvailable || fresh == nil {
			return fmt.Errorf("velopack: no update available")
		}
		info = fresh
		u.pending = fresh
	}
	if err := m.DownloadUpdates(info, progress); err != nil {
		return fmt.Errorf("velopack download failed: %w", err)
	}
	// Restart{} relaunches with no extra args after the update is applied.
	if err := m.WaitForExitThenApplyUpdates(info, velopack.Restart{}); err != nil {
		return fmt.Errorf("velopack apply failed: %w", err)
	}
	return nil
}
