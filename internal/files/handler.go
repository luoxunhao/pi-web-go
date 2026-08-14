package files

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	maxUploadFileBytes  = 25 << 20
	maxUploadTotalBytes = 100 << 20
	maxUploadRequest    = maxUploadTotalBytes + 1<<20
	textPreviewMaxBytes = 256 << 10
)

var ignoredNames = map[string]bool{
	"node_modules": true, ".git": true, ".next": true, "dist": true,
	"build": true, "__pycache__": true, ".turbo": true, ".cache": true,
	"coverage": true, ".pytest_cache": true, ".mypy_cache": true,
	"target": true, "vendor": true, ".DS_Store": true,
}

var languageByExt = map[string]string{
	"ts": "typescript", "tsx": "typescript", "js": "javascript", "jsx": "javascript",
	"py": "python", "go": "go", "rs": "rust", "java": "java", "c": "c",
	"cpp": "cpp", "h": "c", "cs": "csharp", "html": "html", "css": "css",
	"json": "json", "jsonl": "json", "yaml": "yaml", "yml": "yaml",
	"toml": "toml", "xml": "xml", "md": "markdown", "mdx": "markdown",
	"sh": "bash", "bash": "bash", "sql": "sql", "graphql": "graphql",
	"txt": "text", "env": "bash", "gitignore": "bash",
}

// Handler serves the pi-web compatible /api/files endpoints.
type Handler struct {
	Access *Access
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "*")
	if decoded, err := url.PathUnescape(param); err == nil {
		param = decoded
	}
	filePath := filePathFromParam(param)
	if filePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Path is required"})
		return
	}
	rawType := r.URL.Query().Get("type")
	if rawType == "" {
		rawType = "list"
	}

	if !h.Access.IsAllowed(filePath) {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{"error": "Access denied"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.get(w, r, filePath, rawType)
	case http.MethodPost:
		h.post(w, r, filePath, rawType)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "Method not allowed"})
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, filePath, rawType string) {
	switch rawType {
	case "list":
		h.list(w, filePath)
	case "read":
		h.read(w, r, filePath)
	case "download":
		h.download(w, r, filePath)
	case "meta":
		h.meta(w, filePath)
	case "preview":
		h.preview(w, filePath)
	case "watch":
		h.watch(w, r, filePath)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Invalid file request type"})
	}
}

func (h *Handler) post(w http.ResponseWriter, r *http.Request, directory, rawType string) {
	switch rawType {
	case "upload-check":
		h.uploadCheck(w, r, directory)
	case "upload", "":
		h.upload(w, r, directory)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Invalid upload request type"})
	}
}

func (h *Handler) list(w http.ResponseWriter, filePath string) {
	stat, err := os.Stat(filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Not found"})
		return
	}
	if !stat.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Not a directory"})
		return
	}
	entries, err := os.ReadDir(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	out := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		if ignoredNames[e.Name()] || strings.HasSuffix(e.Name(), ".pyc") {
			continue
		}
		out = append(out, map[string]interface{}{"name": e.Name(), "isDir": e.IsDir(), "size": 0, "modified": ""})
	}
	sort.Slice(out, func(i, j int) bool {
		di, _ := out[i]["isDir"].(bool)
		dj, _ := out[j]["isDir"].(bool)
		if di != dj {
			return di
		}
		return out[i]["name"].(string) < out[j]["name"].(string)
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"entries": out, "path": filePath})
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request, filePath string) {
	stat, err := os.Stat(filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Not found"})
		return
	}
	if !stat.Mode().IsRegular() {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Not a file"})
		return
	}
	contentType := mimeType(filePath)
	if contentType != "text/plain" {
		f, err := os.Open(filePath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
			return
		}
		defer f.Close()
		http.ServeContent(w, r, filepath.Base(filePath), stat.ModTime(), f)
		return
	}
	if stat.Size() > textPreviewMaxBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{"error": "File too large for preview (>256KB)"})
		return
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"content":  string(data),
		"language": language(filePath),
		"size":     stat.Size(),
	})
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request, filePath string) {
	stat, err := os.Stat(filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Not found"})
		return
	}
	if !stat.Mode().IsRegular() {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Not a file"})
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(filePath)+`"`)
	f, err := os.Open(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	defer f.Close()
	http.ServeContent(w, r, filepath.Base(filePath), stat.ModTime(), f)
}

func (h *Handler) meta(w http.ResponseWriter, filePath string) {
	stat, err := os.Stat(filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Not found"})
		return
	}
	if !stat.Mode().IsRegular() {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Not a file"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"size":        stat.Size(),
		"language":    language(filePath),
		"mime":        mimeType(filePath),
		"previewKind": previewKind(filePath),
	})
}

func (h *Handler) preview(w http.ResponseWriter, filePath string) {
	if strings.ToLower(filepath.Ext(filePath)) != ".docx" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Preview not available for this file type"})
		return
	}
	stat, err := os.Stat(filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Not found"})
		return
	}
	if stat.Size() > 10<<20 {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{"error": "DOCX too large for preview (>10MB)"})
		return
	}
	html, err := docxPreviewHTML(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Security-Policy", docxCSP)
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.WriteString(w, html)
}

func (h *Handler) watch(w http.ResponseWriter, r *http.Request, filePath string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "streaming unsupported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"filePath\":%q}\n\n", filePath)
	flusher.Flush()

	last := fileSnapshot(filePath)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			cur := fileSnapshot(filePath)
			if cur != last {
				last = cur
				_, _ = fmt.Fprintf(w, "event: change\ndata: {\"mtime\":%q,\"size\":%d}\n\n", cur.modTime, cur.size)
				flusher.Flush()
			}
		}
	}
}

func (h *Handler) uploadCheck(w http.ResponseWriter, r *http.Request, directory string) {
	var body struct {
		FileNames []string `json:"fileNames"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "fileNames must be an array of strings"})
		return
	}
	for _, name := range body.FileNames {
		if !validUploadName(name) {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Invalid file name"})
			return
		}
	}
	conflicts := []string{}
	for _, name := range body.FileNames {
		if _, err := os.Stat(filepath.Join(directory, name)); err == nil {
			conflicts = append(conflicts, name)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"conflicts": conflicts, "nonReplaceable": []string{}})
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request, directory string) {
	stat, err := os.Stat(directory)
	if err != nil || !stat.IsDir() {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Upload directory not found"})
		return
	}
	strategy := r.URL.Query().Get("conflict")
	if strategy == "" {
		strategy = "error"
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadRequest)
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{"error": "Uploads must total 100MB or less"})
		return
	}
	defer r.MultipartForm.RemoveAll()

	uploaded := []string{}
	skipped := []string{}
	errors := []map[string]interface{}{}
	total := int64(0)
	for _, headers := range r.MultipartForm.File {
		for _, fh := range headers {
			if fh.Size > maxUploadFileBytes {
				errors = append(errors, map[string]interface{}{"name": fh.Filename, "error": "Each upload must be 25MB or smaller"})
				continue
			}
			total += fh.Size
			if total > maxUploadTotalBytes {
				errors = append(errors, map[string]interface{}{"name": fh.Filename, "error": "Uploads must total 100MB or less"})
				continue
			}
			if !validUploadName(fh.Filename) {
				errors = append(errors, map[string]interface{}{"name": fh.Filename, "error": "Invalid file name"})
				continue
			}
			dest := filepath.Join(directory, fh.Filename)
			if _, err := os.Stat(dest); err == nil {
				if strategy == "skip" {
					skipped = append(skipped, fh.Filename)
					continue
				}
				if strategy == "error" {
					errors = append(errors, map[string]interface{}{"name": fh.Filename, "error": "One or more files already exist"})
					continue
				}
			}
			src, err := fh.Open()
			if err != nil {
				errors = append(errors, map[string]interface{}{"name": fh.Filename, "error": err.Error()})
				continue
			}
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				_ = src.Close()
				errors = append(errors, map[string]interface{}{"name": fh.Filename, "error": err.Error()})
				continue
			}
			_, copyErr := io.Copy(out, src)
			_ = src.Close()
			_ = out.Close()
			if copyErr != nil {
				errors = append(errors, map[string]interface{}{"name": fh.Filename, "error": copyErr.Error()})
				continue
			}
			uploaded = append(uploaded, fh.Filename)
		}
	}
	status := http.StatusOK
	if len(errors) > 0 {
		status = http.StatusMultiStatus
	}
	writeJSON(w, status, map[string]interface{}{"uploaded": uploaded, "skipped": skipped, "errors": errors})
}

type snapshot struct {
	modTime string
	size    int64
}

func fileSnapshot(path string) snapshot {
	stat, err := os.Stat(path)
	if err != nil {
		return snapshot{modTime: "gone", size: 0}
	}
	return snapshot{modTime: stat.ModTime().UTC().Format(time.RFC3339Nano), size: stat.Size()}
}

func filePathFromParam(param string) string {
	if param == "" {
		return ""
	}
	p := strings.ReplaceAll(param, "\\", "/")
	if !windowsAbs(p) {
		p = "/" + strings.TrimPrefix(p, "/")
	}
	return filepath.FromSlash(p)
}

func windowsAbs(p string) bool {
	return len(p) >= 3 && ((p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= 'a' && p[0] <= 'z')) && p[1] == ':' && (p[2] == '/' || p[2] == '\\')
}

func validUploadName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`) && !strings.Contains(name, "\x00")
}

func language(path string) string {
	base := strings.ToLower(filepath.Base(path))
	if base == "dockerfile" || strings.HasPrefix(base, "dockerfile.") {
		return "dockerfile"
	}
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return "bash"
	}
	ext := strings.TrimPrefix(filepath.Ext(base), ".")
	if lang, ok := languageByExt[ext]; ok {
		return lang
	}
	return "text"
}

func mimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return mime.TypeByExtension(ext)
	case ".mp3", ".wav", ".ogg", ".m4a":
		return mime.TypeByExtension(ext)
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "text/plain"
	}
}

func previewKind(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx":
		return "docx"
	default:
		return ""
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
