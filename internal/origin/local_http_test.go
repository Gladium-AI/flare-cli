package origin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalHTTPStart(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	o, err := NewLocalHTTP(Config{
		Type: TypeLocalHTTP,
		URL:  ts.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	url, err := o.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if url != ts.URL {
		t.Errorf("expected %s, got %s", ts.URL, url)
	}

	if err := o.Healthy(context.Background()); err != nil {
		t.Errorf("Healthy should return nil: %v", err)
	}
}

func TestLocalHTTPStartUnreachable(t *testing.T) {
	o, err := NewLocalHTTP(Config{
		Type:         TypeLocalHTTP,
		URL:          "http://127.0.0.1:1", // nothing should be on port 1
		WaitForReady: "500ms",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = o.Start(context.Background())
	if err == nil {
		t.Error("expected error for unreachable URL")
	}
}

func TestLocalHTTPHealthCheckFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	o, err := NewLocalHTTP(Config{
		Type:         TypeLocalHTTP,
		URL:          ts.URL,
		WaitForReady: "500ms",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Start should fail because health check always returns 500.
	_, err = o.Start(context.Background())
	if err == nil {
		t.Error("expected error for 500 status health check")
	}
}

func TestLocalHTTPMissingURL(t *testing.T) {
	_, err := NewLocalHTTP(Config{Type: TypeLocalHTTP})
	if err == nil {
		t.Error("expected error for missing URL")
	}
}
