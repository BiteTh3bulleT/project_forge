package watch

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"forge/projectforge/services/core/internal/events"
	"forge/projectforge/services/core/internal/ingest"
)

type Manager struct {
	mu sync.Mutex

	watcher *fsnotify.Watcher
	ingest  *ingest.Service
	log     *events.Logger

	debounce   time.Duration
	debounceAt *time.Timer

	watched map[string]struct{}
}

func New(ing *ingest.Service, log *events.Logger) (*Manager, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Manager{
		watcher:  w,
		ingest:   ing,
		log:      log,
		debounce: 700 * time.Millisecond,
		watched:  map[string]struct{}{},
	}, nil
}

func (m *Manager) Run(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-m.watcher.Events:
				if !ok {
					return
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
					continue
				}
				m.scheduleReindex()
			case err, ok := <-m.watcher.Errors:
				if !ok {
					return
				}
				_ = m.log.Emit(context.Background(), "error.raised", map[string]any{"where": "fsnotify", "message": err.Error()})
			}
		}
	}()
}

func (m *Manager) scheduleReindex() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.debounceAt != nil {
		m.debounceAt.Stop()
	}
	m.debounceAt = time.AfterFunc(m.debounce, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := m.ingest.IndexAllSources(ctx); err != nil {
			log.Printf("watch reindex all: %v", err)
		}
	})
}

func (m *Manager) SyncSources(ctx context.Context, paths []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	next := map[string]struct{}{}
	for _, p := range paths {
		next[p] = struct{}{}
	}
	for p := range m.watched {
		if _, ok := next[p]; !ok {
			_ = m.watcher.Remove(p)
			delete(m.watched, p)
		}
	}
	for p := range next {
		if _, ok := m.watched[p]; ok {
			continue
		}
		if err := m.watcher.Add(p); err != nil {
			_ = m.log.Emit(ctx, "error.raised", map[string]any{"where": "watch.add", "path": p, "message": err.Error()})
			continue
		}
		m.watched[p] = struct{}{}
	}
	return nil
}

func (m *Manager) Close() error {
	if m.watcher == nil {
		return nil
	}
	return m.watcher.Close()
}
