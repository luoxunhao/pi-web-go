// Package session tracks the running/queued state of pigo sessions on the Go
// server. pigo's /status endpoint currently hardcodes "idle" (issue 02 G2),
// so state is event-driven and reconciled from pigo session listings.
package session

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

type State struct {
	SessionID string
	Directory string
	Running   bool
	Queued    int
	MessageID string
	LastSeen  time.Time
}

type Manager struct {
	mu          sync.Mutex
	states      map[string]*State
	directories map[string]string
	subs        map[chan []string]struct{}
	idleTimeout time.Duration
}

func NewManager(idleTimeout time.Duration) *Manager {
	if idleTimeout <= 0 {
		idleTimeout = 10 * time.Minute
	}
	return &Manager{
		states:      make(map[string]*State),
		directories: make(map[string]string),
		subs:        make(map[chan []string]struct{}),
		idleTimeout: idleTimeout,
	}
}

// SetDirectory records the pigo directory for a session so later agent
// commands can build pigo requests without re-deriving the cwd.
func (m *Manager) SetDirectory(sessionID, directory string) {
	if sessionID == "" || directory == "" {
		return
	}
	m.mu.Lock()
	m.directories[sessionID] = directory
	if st, ok := m.states[sessionID]; ok {
		st.Directory = directory
	}
	m.mu.Unlock()
}

func (m *Manager) Directory(sessionID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if dir, ok := m.directories[sessionID]; ok {
		return dir
	}
	if st, ok := m.states[sessionID]; ok {
		return st.Directory
	}
	return ""
}

// ObserveEvent updates state from a pigo SSE domain event and publishes the
// running set when it changes.
func (m *Manager) ObserveEvent(ev pigo.DomainEvent) {
	sessionID, _ := ev.Data["sessionId"].(string)
	if sessionID == "" {
		return
	}
	directory, _ := ev.Data["directory"].(string)
	messageID, _ := ev.Data["messageId"].(string)

	m.mu.Lock()
	st := m.stateLocked(sessionID, directory)
	wasRunning := st.Running
	now := time.Now()
	st.LastSeen = now
	if messageID != "" {
		st.MessageID = messageID
	}

	switch ev.Type {
	case "session.status":
		status, _ := ev.Data["status"].(string)
		switch status {
		case "running", "compacting":
			st.Running = true
		case "idle", "cancelled", "error":
			st.Running = false
			st.Queued = 0
		}
	case "queue.updated":
		if n, ok := toInt(ev.Data["queuedCount"]); ok {
			st.Queued = n
			if n > 0 {
				st.Running = true
			}
		}
	case "message.part.delta", "tool.updated", "permission.asked":
		st.Running = true
	}
	changed := st.Running != wasRunning
	ids := m.runningIDsLocked()
	m.mu.Unlock()
	if changed {
		m.publish(ids)
	}
}

// MarkRunning is used by non-SSE paths (for example prompt submission) when a
// run is known to have started.
func (m *Manager) MarkRunning(sessionID, directory, messageID string) {
	m.mu.Lock()
	st := m.stateLocked(sessionID, directory)
	st.Running = true
	st.LastSeen = time.Now()
	if messageID != "" {
		st.MessageID = messageID
	}
	ids := m.runningIDsLocked()
	m.mu.Unlock()
	m.publish(ids)
}

// MarkIdle forces a session back to idle.
func (m *Manager) MarkIdle(sessionID string) {
	m.mu.Lock()
	st, ok := m.states[sessionID]
	if !ok {
		m.mu.Unlock()
		return
	}
	st.Running = false
	st.Queued = 0
	ids := m.runningIDsLocked()
	m.mu.Unlock()
	m.publish(ids)
}

func (m *Manager) RunningIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runningIDsLocked()
}

func (m *Manager) Snapshot() []State {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]State, 0, len(m.states))
	for _, st := range m.states {
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SessionID < out[j].SessionID })
	return out
}

// Cleanup marks sessions inactive when they have not been seen for the idle
// timeout.
func (m *Manager) Cleanup(now time.Time) {
	m.mu.Lock()
	changed := false
	for _, st := range m.states {
		if st.Running && now.Sub(st.LastSeen) > m.idleTimeout {
			st.Running = false
			st.Queued = 0
			changed = true
		}
	}
	ids := m.runningIDsLocked()
	m.mu.Unlock()
	if changed {
		m.publish(ids)
	}
}

// Reconcile lists sessions from pigo so state survives a Go server restart or
// a missed event. Running flags are not guessed from /status (issue 02 G2);
// listing only rehydrates known sessions and drops deleted ones.
func (m *Manager) Reconcile(ctx context.Context, client *pigo.Client, directories []string) error {
	m.mu.Lock()
	known := make(map[string]bool, len(m.states))
	for id := range m.states {
		known[id] = true
	}
	m.mu.Unlock()

	seen := make(map[string]bool)
	listedAny := false
	for _, dir := range directories {
		list, err := client.ListSessions(ctx, dir, 200)
		if err != nil {
			continue
		}
		listedAny = true
		for _, s := range list.Sessions {
			seen[s.SessionID] = true
			m.mu.Lock()
			st := m.stateLocked(s.SessionID, s.Directory)
			st.LastSeen = time.Now()
			m.mu.Unlock()
		}
	}

	m.mu.Lock()
	if listedAny {
		for id := range m.states {
			if !seen[id] {
				delete(m.states, id)
			}
		}
	}
	ids := m.runningIDsLocked()
	m.mu.Unlock()
	m.publish(ids)
	return nil
}

// Subscribe returns a channel receiving the current running set whenever it
// changes, plus an unsubscribe function.
func (m *Manager) Subscribe() (<-chan []string, func()) {
	ch := make(chan []string, 4)
	m.mu.Lock()
	m.subs[ch] = struct{}{}
	m.mu.Unlock()
	return ch, func() {
		m.mu.Lock()
		delete(m.subs, ch)
		m.mu.Unlock()
	}
}

func (m *Manager) stateLocked(sessionID, directory string) *State {
	st, ok := m.states[sessionID]
	if !ok {
		st = &State{SessionID: sessionID, Directory: directory}
		m.states[sessionID] = st
	}
	if st.Directory == "" && directory != "" {
		st.Directory = directory
		m.directories[sessionID] = directory
	}
	return st
}

func (m *Manager) runningIDsLocked() []string {
	ids := make([]string, 0, len(m.states))
	for id, st := range m.states {
		if st.Running {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (m *Manager) publish(ids []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for ch := range m.subs {
		select {
		case ch <- ids:
		default:
		}
	}
}

func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
