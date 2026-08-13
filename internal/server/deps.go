package server

import (
	"github.com/luoxunhao/pi-web-go/internal/events"
	"github.com/luoxunhao/pi-web-go/internal/files"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
	"github.com/luoxunhao/pi-web-go/internal/session"
)

// Dependencies carries the shared services used by the HTTP handlers.
type Dependencies struct {
	PigoClient   *pigo.Client
	Converter    *events.Converter
	Cursor       *events.CursorStore
	SessionMgr   *session.Manager
	FileAccess   *files.Access
	StaticDir    string
	WebPassword  string
	AllowedHosts []string
}
