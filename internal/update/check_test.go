package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func newTestClient(srv *httptest.Server) *Client {
	return &Client{Endpoint: srv.URL, HTTPClient: srv.Client()}
}

func TestLatestRelease(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Write([]byte(`{"tag_name":"v0.2.0","html_url":"https://github.com/Shuro/Gobchat-Log-Browser/releases/tag/v0.2.0"}`))
	}))
	defer srv.Close()

	rel, err := newTestClient(srv).LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.TagName != "v0.2.0" {
		t.Errorf("TagName = %q; want v0.2.0", rel.TagName)
	}
	if !strings.HasSuffix(rel.HTMLURL, "/v0.2.0") {
		t.Errorf("HTMLURL = %q; want release page URL", rel.HTMLURL)
	}
	// GitHub rejects requests without a User-Agent, so the client must send one.
	if !strings.HasPrefix(gotUA, "Gobchat-Log-Browser/") {
		t.Errorf("User-Agent = %q; want Gobchat-Log-Browser/<version>", gotUA)
	}
}

func TestLatestReleaseErrorStatus(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		_, err := newTestClient(srv).LatestRelease(context.Background())
		srv.Close()
		if err == nil {
			t.Fatalf("status %d: want error, got nil", status)
		}
		if !strings.Contains(err.Error(), strconv.Itoa(status)) {
			t.Errorf("status %d: error %q does not mention the status", status, err)
		}
	}
}

func TestLatestReleaseMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{not json`))
	}))
	defer srv.Close()

	if _, err := newTestClient(srv).LatestRelease(context.Background()); err == nil {
		t.Fatal("want error for malformed JSON, got nil")
	}
}

func TestLatestReleaseTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := c.LatestRelease(ctx); err == nil {
		t.Fatal("want timeout error, got nil")
	}
}
