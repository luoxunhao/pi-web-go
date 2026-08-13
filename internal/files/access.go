// Package files implements the pi-web compatible file allow-list and file
// endpoints. The allow-list is owned by pi-web-go, not pigo.
package files

import (
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// Access is a thread-safe allow-list of directory roots. Paths are checked
// lexically and, when the target exists, against resolved symlinks.
type Access struct {
	mu    sync.RWMutex
	roots map[string]struct{}
}

func NewAccess(roots []string) *Access {
	a := &Access{roots: make(map[string]struct{}, len(roots))}
	for _, root := range roots {
		if root != "" {
			_ = a.Add(root)
		}
	}
	return a
}

func (a *Access) Add(root string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	a.mu.Lock()
	a.roots[normalize(abs)] = struct{}{}
	a.mu.Unlock()
	return nil
}

func (a *Access) Roots() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]string, 0, len(a.roots))
	for root := range a.roots {
		out = append(out, root)
	}
	sort.Strings(out)
	return out
}

// IsAllowed reports whether target is inside any allow-listed root. Existing
// paths are resolved so a symlink cannot escape the allow-list.
func (a *Access) IsAllowed(target string) bool {
	abs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	targetNorm := normalize(abs)

	a.mu.RLock()
	defer a.mu.RUnlock()
	for root := range a.roots {
		if pathWithin(root, targetNorm) {
			return true
		}
	}
	return false
}

func pathWithin(root, target string) bool {
	if target == root {
		return true
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(target, prefix)
}

func normalize(p string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(filepath.Clean(p))
	}
	return filepath.Clean(p)
}
