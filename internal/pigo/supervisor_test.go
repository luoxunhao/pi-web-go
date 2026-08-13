package pigo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luoxunhao/pi-web-go/internal/config"
)

func TestSupervisorReusesHealthyExternal(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"healthy":true}`))
	}))
	defer fake.Close()
	cfg := config.PigoConfig{BaseURL: fake.URL, AutoStart: false}
	s := NewSupervisor(cfg)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if s.BaseURL() != fake.URL {
		t.Fatalf("baseURL = %q", s.BaseURL())
	}
}

func TestRandomPasswordNotEmpty(t *testing.T) {
	if got := randomPassword(); len(got) < 32 {
		t.Fatalf("password too short: %q", got)
	}
}
