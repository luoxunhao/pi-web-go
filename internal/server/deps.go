package server

import (
	"github.com/luoxunhao/pi-web-go/internal/events"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

// Dependencies carries the shared services used by the HTTP handlers.
type Dependencies struct {
	PigoClient   *pigo.Client
	Converter    *events.Converter
	Cursor       *events.CursorStore
	WebPassword  string
	AllowedHosts []string
}
