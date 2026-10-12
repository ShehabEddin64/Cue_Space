package store

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch calls onChange (debounced) whenever a .cue.json file in the folder is
// created, edited, renamed or removed, including by other programs. It returns
// when ctx is cancelled.
func (s *Store) Watch(ctx context.Context, onChange func()) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := w.Add(s.dir); err != nil {
		w.Close()
		return err
	}
	go func() {
		defer w.Close()
		var timer *time.Timer
		for {
			select {
			case <-ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if !strings.HasSuffix(filepath.Base(ev.Name), ext) {
					continue
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(150*time.Millisecond, onChange)
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return nil
}
