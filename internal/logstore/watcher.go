package logstore

import (
	"github.com/fsnotify/fsnotify"
)

// WatchEvent classifies a filesystem change reported to the watcher callback.
type WatchEvent int

const (
	WatchEventCreated WatchEvent = iota
	WatchEventModified
	WatchEventRemoved
)

// Watcher observes one or more directories for log file changes and invokes a
// callback. It only reads change notifications; it never modifies any file
// (read-only guarantee, docs/adr/0007).
type Watcher struct {
	fsw      *fsnotify.Watcher
	onChange func(path string, event WatchEvent)
	done     chan struct{}
}

// Watch starts watching the given directories. The callback is invoked for each
// relevant *.log change. Call Close to stop. The callback runs on the watcher's
// own goroutine, so it must be safe for concurrent use and should not block.
func Watch(dirs []string, onChange func(path string, event WatchEvent)) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		// Skip directories that don't exist yet rather than failing entirely.
		_ = fsw.Add(dir)
	}
	w := &Watcher{fsw: fsw, onChange: onChange, done: make(chan struct{})}
	go w.loop()
	return w, nil
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if !IsLogFile(ev.Name) || w.onChange == nil {
				continue
			}
			switch {
			case ev.Op&fsnotify.Create != 0:
				w.onChange(ev.Name, WatchEventCreated)
			case ev.Op&fsnotify.Write != 0:
				w.onChange(ev.Name, WatchEventModified)
			case ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0:
				w.onChange(ev.Name, WatchEventRemoved)
			}
		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			// Errors are non-fatal for a desktop watcher; ignore and continue.
		}
	}
}

// Close stops the watcher and releases its resources.
func (w *Watcher) Close() error {
	close(w.done)
	return w.fsw.Close()
}
