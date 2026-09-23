package itunes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const sampleResponse = `{
  "resultCount": 1,
  "results": [{
    "wrapperType": "track",
    "collectionId": 1234567,
    "collectionName": "Test Podcast",
    "artistName": "Test Artist",
    "feedUrl": "https://example.com/feed.xml",
    "artworkUrl600": "https://example.com/600.jpg",
    "collectionExplicitness": "notExplicit",
    "genres": ["Technology"]
  }]
}`

// newTestService returns a client pointed at a server that records the query
// it was called with.
func newTestService(t *testing.T, handler http.HandlerFunc) (*ItunesApiServices, func() url.Values) {
	t.Helper()

	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	svc := NewItunesApiServices(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	return svc, func() url.Values { return got }
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript")
	_, _ = w.Write([]byte(sampleResponse))
}

func TestSearchWithContextSendsParams(t *testing.T) {
	svc, query := newTestService(t, okHandler)

	res, err := svc.SearchWithContext(context.Background(), SearchParams{
		Term:    "hardcore history",
		Limit:   25,
		Country: "GB",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"term":    "hardcore history",
		"entity":  "podcast",
		"limit":   "25",
		"country": "GB",
	}
	for k, v := range want {
		if got := query().Get(k); got != v {
			t.Errorf("query %q = %q, want %q", k, got, v)
		}
	}

	if res.ResultCount != 1 {
		t.Fatalf("ResultCount = %d, want 1", res.ResultCount)
	}
	if res.Results[0].FeedURL != "https://example.com/feed.xml" {
		t.Errorf("FeedURL = %q, want https://example.com/feed.xml", res.Results[0].FeedURL)
	}
}

func TestSearchParamsOmitsUnsetAndOutOfRangeValues(t *testing.T) {
	cases := map[string]struct {
		params    SearchParams
		wantLimit string
	}{
		"unset limit":     {SearchParams{Term: "x"}, ""},
		"limit too large": {SearchParams{Term: "x", Limit: 500}, ""},
		"negative limit":  {SearchParams{Term: "x", Limit: -1}, ""},
		"valid limit":     {SearchParams{Term: "x", Limit: 200}, "200"},
	}
	for name, tc := range cases {
		got := tc.params.values()
		if got.Get("limit") != tc.wantLimit {
			t.Errorf("%s: limit = %q, want %q", name, got.Get("limit"), tc.wantLimit)
		}
		if got.Get("entity") != "podcast" {
			t.Errorf("%s: entity = %q, want podcast (the default)", name, got.Get("entity"))
		}
		if _, ok := got["country"]; ok {
			t.Errorf("%s: country should be omitted when unset", name)
		}
	}
}

func TestFindByIdWithContext(t *testing.T) {
	svc, query := newTestService(t, okHandler)

	if _, err := svc.FindByIdWithContext(context.Background(), 1234567); err != nil {
		t.Fatal(err)
	}
	if got := query().Get("id"); got != "1234567" {
		t.Errorf("query id = %q, want 1234567", got)
	}
}

func TestNon2xxReturnsHTTPErrorNotDecodeError(t *testing.T) {
	svc, _ := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		// What iTunes actually serves when it throttles you: an HTML page.
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html><body>Forbidden</body></html>"))
	})

	_, err := svc.SearchWithContext(context.Background(), SearchParams{Term: "x"})
	if err == nil {
		t.Fatal("expected an error for a 403 response")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("error is %T (%v), want *HTTPError", err, err)
	}
	if httpErr.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d, want %d", httpErr.StatusCode, http.StatusForbidden)
	}
}

func TestContextCancellationIsHonoured(t *testing.T) {
	svc, _ := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := svc.SearchWithContext(ctx, SearchParams{Term: "x"}); err == nil {
		t.Fatal("expected an error from a cancelled context")
	}
}
