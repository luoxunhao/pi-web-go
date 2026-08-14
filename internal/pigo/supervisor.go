package pigo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/luoxunhao/pi-web-go/internal/config"
)

// Supervisor owns the pigo serve child process (P2/D1/S1). When auto_start is
// disabled it still verifies that an external pigo is healthy.
type Supervisor struct {
	cfg          config.PigoConfig
	password     string
	baseURL      string
	mu           sync.Mutex
	cmd          *exec.Cmd
	stopped      bool
	managed      bool
	restartDelay time.Duration
	healthClient *http.Client
}

func NewSupervisor(cfg config.PigoConfig) *Supervisor {
	password := cfg.Password
	if password == "" {
		password = randomPassword()
	}
	host := cfg.Host
	if host == "" {
		host = config.DefaultPigoHost
	}
	port := cfg.Port
	if port == 0 {
		port = config.DefaultPigoPort
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", host, port)
	}
	return &Supervisor{
		cfg:          cfg,
		password:     password,
		baseURL:      strings.TrimRight(baseURL, "/"),
		restartDelay: 2 * time.Second,
		healthClient: &http.Client{Timeout: 2 * time.Second},
	}
}

func (s *Supervisor) BaseURL() string {
	return s.baseURL
}

func (s *Supervisor) Password() string {
	return s.password
}

// Start ensures pigo is reachable, spawning it when auto_start is enabled.
func (s *Supervisor) Start(ctx context.Context) error {
	if s.healthy(ctx) {
		return nil
	}
	if !s.cfg.AutoStart {
		return fmt.Errorf("pigo is not healthy at %s and auto_start is disabled", s.baseURL)
	}
	if err := s.checkPortFree(); err != nil {
		return err
	}
	if err := s.spawn(); err != nil {
		return err
	}
	s.mu.Lock()
	s.managed = true
	s.mu.Unlock()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			_ = s.Stop()
			return ctx.Err()
		}
		if s.healthy(ctx) {
			go s.monitor()
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	_ = s.Stop()
	return fmt.Errorf("pigo did not become healthy at %s", s.baseURL)
}

func (s *Supervisor) Stop() error {
	s.mu.Lock()
	s.stopped = true
	cmd := s.cmd
	s.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
	return nil
}

func (s *Supervisor) spawn() error {
	args := append([]string{}, s.cfg.Args...)
	args = append(args,
		"--hostname", s.cfg.Host,
		"--port", strconv.Itoa(s.cfg.Port),
		"--password", s.password,
	)
	cmd := exec.Command(s.cfg.Command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if s.cfg.DataDir != "" {
		cmd.Env = append(os.Environ(), "PIGO_HOME="+s.cfg.DataDir)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pigo: %w", err)
	}
	s.mu.Lock()
	s.cmd = cmd
	s.stopped = false
	s.mu.Unlock()
	return nil
}

func (s *Supervisor) monitor() {
	_, _ = s.cmd.Process.Wait()
	s.mu.Lock()
	stopped := s.stopped
	s.cmd = nil
	s.mu.Unlock()
	if stopped {
		return
	}
	time.Sleep(s.restartDelay)
	s.mu.Lock()
	stopped = s.stopped
	s.mu.Unlock()
	if stopped {
		return
	}
	if err := s.spawn(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "pigo restart failed: %v\n", err)
		return
	}
	go s.monitor()
}

func (s *Supervisor) healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/api/v1/health", nil)
	if err != nil {
		return false
	}
	if s.password != "" {
		req.SetBasicAuth("pigo", s.password)
	}
	resp, err := s.healthClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK
}

func (s *Supervisor) checkPortFree() error {
	host := s.cfg.Host
	if host == "" {
		host = config.DefaultPigoHost
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(s.cfg.Port)))
	if err != nil {
		return fmt.Errorf("pigo port %d is already in use: %w", s.cfg.Port, err)
	}
	_ = ln.Close()
	return nil
}

func randomPassword() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
