//go:build e2e

// Package e2e is a black-box end-to-end suite for the railway-wiki server.
//
// TestMain builds the server binary and launches it as a subprocess against a
// local file SQLite database and (when configured) a dockerized MinIO S3
// backend, then drives it over HTTP through the generated client. Tests carry
// the `e2e` build tag so the default `go test ./...` does not pick them up.
//
// Run with:  make test-e2e      (boots MinIO + exports env, then runs this)
//
//	or:  docker compose up -d minio createbuckets && \
//	     go test -tags e2e -count=1 ./e2e-tests/...
package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rxtech-lab/railway-wiki/e2e-tests/client"
)

// harness holds the resolved e2e configuration and the running server process.
type harness struct {
	baseURL     string
	bearerToken string // OAuth access token for management endpoints
	s3Bucket    string // empty => media/S3 tests skip

	oauth   *oauthProvider // in-process OIDC provider (nil when targeting an external server)
	cmd     *exec.Cmd
	dbPath  string // underlying SQLite file to clean up (empty if not a file DSN)
	workDir string // temp working dir for the child process
}

// h is the shared harness for all tests in this package, initialized in TestMain.
var h *harness

func TestMain(m *testing.M) {
	var err error
	h, err = startHarness()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e harness setup failed: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	h.stop()
	os.Exit(code)
}

// envOr returns the environment value for key, or def when unset/empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// startHarness resolves config, launches the server, and waits until it is healthy.
func startHarness() (*harness, error) {
	port := envOr("PORT", "8080")
	h := &harness{
		baseURL:  envOr("E2E_BASE_URL", "http://localhost:"+port),
		s3Bucket: os.Getenv("S3_BUCKET"),
	}

	// Point at an already-running server if asked; skip building/launching. The
	// external server owns its own OAuth config, so the caller supplies a token.
	if os.Getenv("E2E_BASE_URL") != "" {
		h.bearerToken = os.Getenv("E2E_BEARER_TOKEN")
		if err := waitForHealth(h.baseURL, 30*time.Second); err != nil {
			return nil, fmt.Errorf("server at %s never became healthy: %w", h.baseURL, err)
		}
		return h, nil
	}

	// Stand up an in-process OIDC provider: it serves the JWKS the child server
	// verifies against and mints the admin access token the tests present.
	oauth, err := newOAuthProvider()
	if err != nil {
		return nil, fmt.Errorf("start oauth provider: %w", err)
	}
	h.oauth = oauth
	h.bearerToken, err = oauth.token("e2e-admin", "admin")
	if err != nil {
		return nil, fmt.Errorf("mint access token: %w", err)
	}

	root, err := moduleRoot()
	if err != nil {
		return nil, err
	}

	// Isolated temp working dir so a developer's server/.env cannot bleed in.
	h.workDir, err = os.MkdirTemp("", "railway-e2e-")
	if err != nil {
		return nil, fmt.Errorf("create temp workdir: %w", err)
	}

	// Build the server binary.
	binPath := filepath.Join(h.workDir, "server")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/server")
	build.Dir = root
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		return nil, fmt.Errorf("build server: %w", err)
	}

	// Resolve the database DSN to a clean file so migrations run from scratch.
	dbURL := envOr("DATABASE_URL", "file:"+filepath.Join(h.workDir, "railway-e2e.db"))
	h.dbPath = sqliteFilePath(dbURL)
	if h.dbPath != "" {
		// Ensure the parent dir exists (SQLite won't create it) and start fresh.
		if err := os.MkdirAll(filepath.Dir(h.dbPath), 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
		_ = os.Remove(h.dbPath) // start fresh; ignore if absent
	}

	// Launch the server subprocess with an explicit environment.
	h.cmd = exec.Command(binPath)
	h.cmd.Dir = h.workDir
	h.cmd.Env = h.childEnv(port, dbURL)
	h.cmd.Stdout, h.cmd.Stderr = os.Stdout, os.Stderr
	if err := h.cmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}

	if err := waitForHealth(h.baseURL, 30*time.Second); err != nil {
		h.stop()
		return nil, fmt.Errorf("server never became healthy: %w", err)
	}
	return h, nil
}

// childEnv builds the environment for the server subprocess, honoring any values
// already present in the environment and filling in e2e defaults otherwise.
func (h *harness) childEnv(port, dbURL string) []string {
	env := map[string]string{
		"PORT":         port,
		"DATABASE_URL": dbURL,
		"OAUTH_ISSUER": h.oauth.issuer,
	}
	// Pass S3 settings through only when a bucket is configured (else NoopPresigner).
	if h.s3Bucket != "" {
		env["S3_BUCKET"] = h.s3Bucket
		env["S3_REGION"] = envOr("S3_REGION", "us-east-1")
		env["S3_ENDPOINT"] = envOr("S3_ENDPOINT", "http://localhost:9000")
		env["S3_ACCESS_KEY_ID"] = envOr("S3_ACCESS_KEY_ID", "minioadmin")
		env["S3_SECRET_ACCESS_KEY"] = envOr("S3_SECRET_ACCESS_KEY", "minioadmin")
		env["S3_PUBLIC_URL"] = envOr("S3_PUBLIC_URL", "http://localhost:9000/railway-wiki")
		env["S3_PATH_STYLE"] = envOr("S3_PATH_STYLE", "true")
		env["S3_PRESIGN_TTL"] = envOr("S3_PRESIGN_TTL", "15m")
	}

	// Preserve PATH/HOME etc. from the parent, but drop any DATABASE_URL/S3_* /
	// OAUTH_* the parent may carry so our explicit values win.
	out := []string{}
	for _, kv := range os.Environ() {
		key := kv[:strings.IndexByte(kv, '=')]
		if _, overridden := env[key]; overridden {
			continue
		}
		out = append(out, kv)
	}
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

// stop terminates the server and removes temp artifacts.
func (h *harness) stop() {
	if h.cmd != nil && h.cmd.Process != nil {
		_ = h.cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() { _, _ = h.cmd.Process.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = h.cmd.Process.Kill()
		}
	}
	if h.dbPath != "" {
		_ = os.Remove(h.dbPath)
	}
	if h.workDir != "" {
		_ = os.RemoveAll(h.workDir)
	}
	if h.oauth != nil {
		h.oauth.Close()
	}
}

// --- client factories ---

// newClient returns an unauthenticated client (public read endpoints).
func newClient(t *testing.T) *client.ClientWithResponses {
	t.Helper()
	c, err := client.NewClientWithResponses(h.baseURL)
	if err != nil {
		t.Fatalf("build client: %v", err)
	}
	return c
}

// newAuthedClient returns a client that sends the management OAuth access token.
func newAuthedClient(t *testing.T) *client.ClientWithResponses {
	t.Helper()
	c, err := client.NewClientWithResponses(h.baseURL,
		client.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+h.bearerToken)
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("build authed client: %v", err)
	}
	return c
}

// newRoleClient returns a client presenting a freshly minted access token for
// the given role, using the harness's in-process OIDC provider. It skips the
// test when targeting an external server (no provider to mint from).
func newRoleClient(t *testing.T, subject, role string) *client.ClientWithResponses {
	t.Helper()
	if h.oauth == nil {
		t.Skip("no in-process OIDC provider (external server target); cannot mint custom-role token")
	}
	token, err := h.oauth.token(subject, role)
	if err != nil {
		t.Fatalf("mint %q-role token: %v", role, err)
	}
	c, err := client.NewClientWithResponses(h.baseURL,
		client.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("build role client: %v", err)
	}
	return c
}

// requireS3 skips the calling test unless media object storage is configured.
func requireS3(t *testing.T) {
	t.Helper()
	if h.s3Bucket == "" {
		t.Skip("S3_BUCKET not set; skipping media/S3 e2e test (run `make test-e2e` for MinIO)")
	}
}

// --- helpers ---

func waitForHealth(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := strings.TrimRight(baseURL, "/") + "/health"
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timed out after %s: %w", timeout, lastErr)
}

// sqliteFilePath extracts the on-disk path from a `file:...` SQLite DSN, or ""
// for :memory: / remote libSQL URLs where there is no file to clean up.
func sqliteFilePath(dsn string) string {
	if !strings.HasPrefix(dsn, "file:") {
		return ""
	}
	path := strings.TrimPrefix(dsn, "file:")
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	if path == "" || path == ":memory:" {
		return ""
	}
	return path
}

// moduleRoot walks up from the current directory to the dir containing go.mod.
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s upward", dir)
		}
		dir = parent
	}
}
