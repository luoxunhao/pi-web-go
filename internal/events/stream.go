package events

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

// StreamHandler proxies pigo's SSE stream and converts it to pi-web's wire
// format. The client may pass ?after= to request a replay cursor; otherwise
// the in-memory CursorStore resumes from the last event seen for the session.
type StreamHandler struct {
	Client    *pigo.Client
	Cursor    *CursorStore
	Converter *Converter
	Heartbeat time.Duration
}

type frameBatch struct {
	id     int64
	frames [][]byte
}

func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		sessionID = r.URL.Query().Get("sessionId")
	}
	if sessionID == "" {
		http.Error(w, "sessionId required", http.StatusBadRequest)
		return
	}
	directory := r.URL.Query().Get("directory")
	types := r.URL.Query().Get("types")

	if h.Cursor == nil {
		h.Cursor = NewCursorStore()
	}
	if h.Converter == nil {
		h.Converter = NewConverter()
	}
	after := resolveAfter(r, h.Cursor, sessionID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	if _, err := fmt.Fprint(w, ":\n\n"); err != nil {
		return
	}
	if connected, err := Marshal(WireEvent{
		"type":        "connected",
		"sessionId":   sessionID,
		"isStreaming": false,
	}); err == nil {
		_, _ = w.Write(connected)
	}
	flusher.Flush()

	heartbeat := h.Heartbeat
	if heartbeat <= 0 {
		heartbeat = 30 * time.Second
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	batches := make(chan frameBatch, 32)
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.Client.StreamEvents(ctx, sessionID, directory, after, types, func(ev pigo.DomainEvent) error {
			wire := h.Converter.Convert(ev)
			if len(wire) == 0 {
				return nil
			}
			frames := make([][]byte, 0, len(wire))
			for _, item := range wire {
				frame, err := MarshalWithID(item, ev.ID)
				if err != nil {
					return err
				}
				frames = append(frames, frame)
			}
			select {
			case batches <- frameBatch{id: ev.ID, frames: frames}:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()

	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			for {
				select {
				case batch := <-batches:
					for _, frame := range batch.frames {
						if _, writeErr := w.Write(frame); writeErr != nil {
							return
						}
					}
					h.Cursor.Set(sessionID, batch.id)
					flusher.Flush()
				default:
					goto drained
				}
			}
		drained:
			if err != nil && ctx.Err() == nil {
				startup, marshalErr := Marshal(WireEvent{"type": "startup_error", "errorMessage": err.Error()})
				if marshalErr == nil {
					_, _ = w.Write(startup)
				}
				flusher.Flush()
			}
			return
		case batch := <-batches:
			for _, frame := range batch.frames {
				if _, err := w.Write(frame); err != nil {
					return
				}
			}
			h.Cursor.Set(sessionID, batch.id)
			flusher.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ":\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func resolveAfter(r *http.Request, cursor *CursorStore, sessionID string) int64 {
	if v := r.URL.Query().Get("after"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			return n
		}
	}
	if v := r.Header.Get("Last-Event-ID"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			return n
		}
	}
	return cursor.Get(sessionID)
}
