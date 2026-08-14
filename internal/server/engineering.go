package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/luoxunhao/pi-web-go/internal/files"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

type engineeringHandler struct {
	access     *files.Access
	pigoClient *pigo.Client
}

func (h *engineeringHandler) home(w http.ResponseWriter, _ *http.Request) {
	home, _ := os.UserHomeDir()
	modelJSON(w, http.StatusOK, map[string]interface{}{"home": home})
}

func (h *engineeringHandler) cwdBrowse(w http.ResponseWriter, r *http.Request) {
	requested := strings.TrimSpace(r.URL.Query().Get("path"))
	start := requested
	if start == "" || start == "~" {
		home, _ := os.UserHomeDir()
		start = home
	} else if strings.HasPrefix(start, "~/") {
		home, _ := os.UserHomeDir()
		start = filepath.Join(home, strings.TrimPrefix(start, "~/"))
	}
	stat, err := os.Stat(start)
	if err != nil || !stat.IsDir() {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Directory does not exist"})
		return
	}
	entries, err := os.ReadDir(start)
	if err != nil {
		modelJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	dirs := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	parent := filepath.Dir(start)
	if filepath.Clean(parent) == filepath.Clean(start) {
		parent = ""
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{
		"path": start, "parentPath": parent, "directories": dirs,
	})
}

func (h *engineeringHandler) cwdValidate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cwd string `json:"cwd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Cwd) == "" {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Path is required"})
		return
	}
	cwd := body.Cwd
	if cwd == "~" {
		home, _ := os.UserHomeDir()
		cwd = home
	} else if strings.HasPrefix(cwd, "~/") {
		home, _ := os.UserHomeDir()
		cwd = filepath.Join(home, strings.TrimPrefix(cwd, "~/"))
	}
	stat, err := os.Stat(cwd)
	if err != nil || !stat.IsDir() {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Directory does not exist: " + body.Cwd})
		return
	}
	if h.access != nil {
		_ = h.access.Add(cwd)
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"success": true, "cwd": cwd})
}

func (h *engineeringHandler) defaultCwd(w http.ResponseWriter, _ *http.Request) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "pi-cwd-"+time.Now().Format("20060102"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		modelJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	if h.access != nil {
		_ = h.access.Add(dir)
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"cwd": dir})
}

func (h *engineeringHandler) gitStatus(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if !h.allowCwd(w, cwd) {
		return
	}
	root, rootErr := gitOutput(cwd, "rev-parse", "--show-toplevel")
	isGit := rootErr == nil
	files := []map[string]interface{}{}
	additions, deletions := 0, 0
	if isGit {
		if out, err := gitOutput(cwd, "status", "--porcelain=v1", "-z", "--untracked-files=all"); err == nil {
			for _, record := range strings.Split(out, "\x00") {
				if len(record) < 4 || record[2] != ' ' {
					continue
				}
				indexStatus := record[0:1]
				worktreeStatus := record[1:2]
				path := record[3:]
				files = append(files, map[string]interface{}{
					"filePath": path, "status": classifyGit(indexStatus, worktreeStatus),
					"code":        statusCode(indexStatus, worktreeStatus),
					"indexStatus": indexStatus, "worktreeStatus": worktreeStatus,
				})
			}
		}
		if out, err := gitOutput(cwd, "diff", "--numstat"); err == nil {
			for _, line := range strings.Split(out, "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					additions += atoi(fields[0])
					deletions += atoi(fields[1])
				}
			}
		}
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{
		"isGitRepository": isGit, "repositoryRoot": strings.TrimSpace(root),
		"files": files, "additions": additions, "deletions": deletions,
	})
}

func (h *engineeringHandler) gitDiff(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	filePath := r.URL.Query().Get("path")
	if !h.allowCwd(w, cwd) || filePath == "" {
		if filePath == "" {
			modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "path is required"})
		}
		return
	}
	if h.access != nil && !h.access.IsAllowed(filePath) {
		modelJSON(w, http.StatusForbidden, map[string]interface{}{"error": "Access denied"})
		return
	}
	patch, _ := gitOutput(cwd, "diff", "--", filePath)
	if patch == "" {
		patch, _ = gitOutput(cwd, "diff", "--cached", "--", filePath)
	}
	status := "modified"
	if patch == "" {
		status = "untracked"
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"supported": true, "status": status, "patch": patch})
}

func (h *engineeringHandler) worktrees(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if !h.allowCwd(w, cwd) {
		return
	}
	out, err := gitOutput(cwd, "worktree", "list", "--porcelain")
	if err != nil {
		modelJSON(w, http.StatusOK, map[string]interface{}{
			"projectRoot": cwd, "isGit": false, "isTopLevel": true, "currentWorktreePath": nil, "worktrees": []interface{}{},
		})
		return
	}
	worktrees := []map[string]interface{}{}
	var current *string
	for _, block := range strings.Split(out, "\n\n") {
		var path, branch string
		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(line, "worktree ") {
				path = strings.TrimPrefix(line, "worktree ")
			} else if strings.HasPrefix(line, "branch refs/heads/") {
				branch = strings.TrimPrefix(line, "branch refs/heads/")
			}
		}
		if path != "" {
			worktrees = append(worktrees, map[string]interface{}{"path": path, "branch": branch})
			if h.access != nil {
				_ = h.access.Add(path)
			}
			if filepath.Clean(path) == filepath.Clean(cwd) {
				current = &path
			}
		}
	}
	root, _ := gitOutput(cwd, "rev-parse", "--show-toplevel")
	modelJSON(w, http.StatusOK, map[string]interface{}{
		"projectRoot": strings.TrimSpace(root), "isGit": true, "isTopLevel": filepath.Clean(root) == filepath.Clean(cwd),
		"currentWorktreePath": current, "worktrees": worktrees,
	})
}

func (h *engineeringHandler) worktreeAdd(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cwd    string `json:"cwd"`
		Branch string `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Cwd == "" || body.Branch == "" {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cwd and branch are required"})
		return
	}
	if !h.allowCwd(w, body.Cwd) {
		return
	}
	root, _ := gitOutput(body.Cwd, "rev-parse", "--show-toplevel")
	root = strings.TrimSpace(root)
	target := filepath.Join(root+"-worktrees", sanitizeBranch(body.Branch))
	if _, err := gitOutput(body.Cwd, "worktree", "add", target, body.Branch); err != nil {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	if h.access != nil {
		_ = h.access.Add(target)
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"path": target, "branch": body.Branch})
}

func (h *engineeringHandler) worktreeRemove(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cwd   string `json:"cwd"`
		Path  string `json:"path"`
		Force bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Cwd == "" || body.Path == "" {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cwd and path are required"})
		return
	}
	if !h.allowCwd(w, body.Cwd) {
		return
	}
	args := []string{"worktree", "remove"}
	if body.Force {
		args = append(args, "--force")
	}
	args = append(args, body.Path)
	if _, err := gitOutput(body.Cwd, args...); err != nil {
		msg := err.Error()
		dirty := strings.Contains(msg, "modified or untracked") || strings.Contains(msg, "dirty")
		modelJSON(w, map[bool]int{true: http.StatusConflict, false: http.StatusBadRequest}[dirty], map[string]interface{}{"error": msg, "dirty": dirty})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *engineeringHandler) fileIndex(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if !h.allowCwd(w, cwd) {
		return
	}
	query := r.URL.Query().Get("q")
	files, truncated := gitFileList(cwd)
	if query != "" {
		matches := []map[string]interface{}{}
		for _, f := range files {
			if strings.Contains(strings.ToLower(f), strings.ToLower(query)) {
				matches = append(matches, map[string]interface{}{"path": f, "isDir": false})
			}
		}
		modelJSON(w, http.StatusOK, map[string]interface{}{"matches": matches})
		return
	}
	if len(files) > 5000 {
		files = files[:5000]
		truncated = true
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"files": files, "truncated": truncated})
}

func (h *engineeringHandler) bashOutput(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" || h.access == nil || !h.access.IsAllowed(path) {
		modelJSON(w, http.StatusForbidden, map[string]interface{}{"error": "Access denied"})
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "full output unavailable"})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{"output": string(data)}})
}

func (h *engineeringHandler) appUpdate(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("https://registry.npmjs.org/@agegr%2Fpi-web/latest")
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	var body struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{
		"currentVersion": "0.8.8", "latestVersion": body.Version,
		"updateAvailable": body.Version != "" && body.Version != "0.8.8",
		"releaseUrl":      "https://github.com/agegr/pi-web/releases/tag/v" + body.Version,
	})
}

func (h *engineeringHandler) projectTrustGet(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if strings.TrimSpace(cwd) == "" {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cwd is required"})
		return
	}
	if h.access != nil && !h.access.IsAllowed(cwd) {
		modelJSON(w, http.StatusForbidden, map[string]interface{}{"error": "Access denied"})
		return
	}
	trusted := false
	requireTrust := false
	if h.pigoClient != nil {
		result, err := h.pigoClient.ListTrust(r.Context())
		if err == nil {
			for _, e := range result.Entries {
				if filepath.Clean(e.Path) == filepath.Clean(cwd) && e.Trust != nil && *e.Trust {
					trusted = true
					break
				}
			}
			// requiresTrust is only meaningful when trust info is available
			requireTrust = trusted
		}
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{
		"requiresTrust": requireTrust,
		"trusted":       trusted,
	})
}

func (h *engineeringHandler) projectTrustPost(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cwd string `json:"cwd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Cwd) == "" {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cwd is required"})
		return
	}
	cwd := body.Cwd
	if h.access != nil && !h.access.IsAllowed(cwd) {
		modelJSON(w, http.StatusForbidden, map[string]interface{}{"error": "Access denied"})
		return
	}
	if h.pigoClient == nil {
		modelJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"error": "Agent client not available"})
		return
	}
	if err := h.pigoClient.SetTrust(r.Context(), pigo.SetTrustRequest{Path: cwd, Trust: boolPtr(true)}); err != nil {
		modelJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"requiresTrust": true, "trusted": true})
}

func boolPtr(b bool) *bool { return &b }

func (h *engineeringHandler) allowCwd(w http.ResponseWriter, cwd string) bool {
	if cwd == "" {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cwd must be an absolute path"})
		return false
	}
	if h.access != nil && !h.access.IsAllowed(cwd) {
		modelJSON(w, http.StatusForbidden, map[string]interface{}{"error": "Access denied"})
		return false
	}
	if stat, err := os.Stat(cwd); err != nil || !stat.IsDir() {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Directory not found"})
		return false
	}
	return true
}

func gitOutput(cwd string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", cwd}, args...)...)
	out, err := cmd.Output()
	return string(out), err
}

func classifyGit(index, worktree string) string {
	pair := index + worktree
	switch {
	case pair == "??":
		return "untracked"
	case strings.Contains(pair, "U"):
		return "conflict"
	case strings.Contains(pair, "D"):
		return "deleted"
	case strings.Contains(pair, "R") || strings.Contains(pair, "C"):
		return "renamed"
	case strings.Contains(pair, "A"):
		return "added"
	default:
		return "modified"
	}
}

func statusCode(index, worktree string) string {
	switch classifyGit(index, worktree) {
	case "untracked":
		return "U"
	case "conflict":
		return "C"
	case "deleted":
		return "D"
	case "renamed":
		return "R"
	case "added":
		return "A"
	default:
		return "M"
	}
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func sanitizeBranch(branch string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '-'
		}
		return r
	}, branch)
}

func gitFileList(cwd string) ([]string, bool) {
	if out, err := gitOutput(cwd, "ls-files", "--cached", "--others", "--exclude-standard", "-z"); err == nil {
		files := strings.Split(out, "\x00")
		files = files[:len(files)-1]
		return files, len(files) > 200000
	}
	files := []string{}
	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(cwd, path)
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	return files, false
}
