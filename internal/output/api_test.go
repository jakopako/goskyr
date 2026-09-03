package output

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestAPIWriterDeleteUsesRFC3339FromTime(t *testing.T) {
	var deleteQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleteQuery = r.URL.Query()
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			return
		}
		t.Errorf("unexpected method %s", r.Method)
	}))
	defer server.Close()

	firstDate := time.Date(2026, 9, 3, 15, 29, 32, 0, time.FixedZone("CEST", 2*60*60))
	writer, err := NewAPIWriter(&WriterConfig{
		Uri:       server.URL,
		BatchSize: 1,
	})
	if err != nil {
		t.Fatalf("NewAPIWriter() error = %v", err)
	}

	items := make(chan map[string]any, 1)
	items <- map[string]any{
		"sourceUrl": "https://example.com/events",
		"date":      firstDate,
	}
	close(items)
	writer.Write(items)

	if got, want := deleteQuery.Get("fromTime"), firstDate.UTC().Format(time.RFC3339); got != want {
		t.Errorf("fromTime = %q, want %q", got, want)
	}
	if got := deleteQuery.Get("datetime"); got != "" {
		t.Errorf("deprecated datetime query parameter = %q, want it omitted", got)
	}
}
