package endpoint

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

func serve(t *testing.T, filename string, size int, body func(w http.ResponseWriter)) *Provider {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/m.json" {
			fmt.Fprintf(w, `{"schemaVersion":1,"version":"2.0.0","artifacts":[{"url":"/f","filename":%q,"size":%d}]}`, filename, size)
			return
		}
		body(w)
	}))
	t.Cleanup(srv.Close)
	p, err := New(Config{URL: srv.URL + "/m.json"})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCheckRejectsFilenamesWithAPath(t *testing.T) {
	for _, name := range []string{"../x", "a/b", `a\b`, "..", "C:x", "a\tb"} {
		p := serve(t, name, 1, func(http.ResponseWriter) {})
		if _, err := p.Check(context.Background(), checkReq()); err == nil || !strings.Contains(err.Error(), "plain file name") {
			t.Errorf("filename %q: err = %v", name, err)
		}
	}
}

func TestDownloadStopsAtTheDeclaredSize(t *testing.T) {
	p := serve(t, "a.bin", 10, func(w http.ResponseWriter) { w.Write(make([]byte, 5000)) })
	rel, err := p.Check(context.Background(), updater.CheckRequest{CurrentVersion: "1.0.0", Platform: "darwin", Arch: "arm64"})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := p.Download(context.Background(), rel, &buf, nil); err == nil || buf.Len() > 10 {
		t.Fatalf("err = %v, wrote %d", err, buf.Len())
	}
}

func TestDownloadShorterThanDeclaredFails(t *testing.T) {
	p := serve(t, "a.bin", 10, func(w http.ResponseWriter) { w.Write([]byte("abc")) })
	rel, _ := p.Check(context.Background(), checkReq())
	if err := p.Download(context.Background(), rel, &bytes.Buffer{}, nil); err == nil {
		t.Fatal("a short body was accepted")
	}
}

func TestDefaultClientHasNoTotalTimeout(t *testing.T) {
	p, _ := New(Config{URL: "http://x/m.json"})
	if p.client.Timeout != 0 {
		t.Fatalf("Timeout = %s: a large artifact on a slow link would be cut off", p.client.Timeout)
	}
}

func TestStalledDownloadEndsAtTheIdleTimeoutButASlowOneContinues(t *testing.T) {
	old := downloadIdleTimeout
	downloadIdleTimeout = 300 * time.Millisecond
	defer func() { downloadIdleTimeout = old }()

	slow := serve(t, "a.bin", 0, func(w http.ResponseWriter) {
		for range 8 {
			w.Write([]byte("x"))
			w.(http.Flusher).Flush()
			time.Sleep(100 * time.Millisecond)
		}
	})
	rel, _ := slow.Check(context.Background(), checkReq())
	if err := slow.Download(context.Background(), rel, &bytes.Buffer{}, nil); err != nil {
		t.Fatalf("a steady slow download failed: %v", err)
	}

	stall := serve(t, "a.bin", 0, func(w http.ResponseWriter) {
		w.Write([]byte("x"))
		w.(http.Flusher).Flush()
		time.Sleep(3 * time.Second)
	})
	rel, _ = stall.Check(context.Background(), checkReq())
	start := time.Now()
	if err := stall.Download(context.Background(), rel, &bytes.Buffer{}, nil); err == nil || time.Since(start) > 2*time.Second {
		t.Fatalf("stall: err = %v after %s", err, time.Since(start))
	}
}
