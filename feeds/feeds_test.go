package feeds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"
     xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Test Podcast</title>
    <itunes:new-feed-url>https://example.com/new.xml</itunes:new-feed-url>
    <itunes:category text="Technology">
      <itunes:category text="Podcasting"/>
    </itunes:category>
    <itunes:category text="Society &amp; Culture"/>
    <itunes:image href="https://example.com/art.jpg"/>
    <itunes:type>episodic</itunes:type>
    <item>
      <title>Episode One</title>
      <guid isPermaLink="false">ep-1-guid</guid>
      <pubDate>Wed, 01 Jan 2025 00:00:00 +0000</pubDate>
      <enclosure url="https://example.com/1.mp3" length="12345" type="audio/mpeg"/>
      <itunes:duration>01:02:03</itunes:duration>
      <itunes:explicit>false</itunes:explicit>
      <itunes:episode>1</itunes:episode>
      <itunes:season>2</itunes:season>
      <itunes:episodeType>full</itunes:episodeType>
      <itunes:image href="https://example.com/ep1.jpg"/>
      <content:encoded><![CDATA[<p>Full show notes</p>]]></content:encoded>
    </item>
  </channel>
</rss>`

func TestParseExtractsPodcastFields(t *testing.T) {
	rss, err := ParseBytes([]byte(sampleFeed))
	if err != nil {
		t.Fatal(err)
	}

	ch := rss.Channel
	if ch.Title != "Test Podcast" {
		t.Errorf("title = %q, want %q", ch.Title, "Test Podcast")
	}
	if ch.NewFeedURL != "https://example.com/new.xml" {
		t.Errorf("new-feed-url = %q, want https://example.com/new.xml", ch.NewFeedURL)
	}
	if ch.Type != "episodic" {
		t.Errorf("itunes:type = %q, want episodic", ch.Type)
	}
	if got := ch.ImageURL(); got != "https://example.com/art.jpg" {
		t.Errorf("ImageURL() = %q, want https://example.com/art.jpg", got)
	}

	if len(ch.Category) != 2 {
		t.Fatalf("got %d categories, want 2", len(ch.Category))
	}
	if ch.Category[0].AttrText != "Technology" {
		t.Errorf("category[0] = %q, want Technology", ch.Category[0].AttrText)
	}
	if len(ch.Category[0].Sub) != 1 || ch.Category[0].Sub[0].AttrText != "Podcasting" {
		t.Errorf("category[0] subcategories = %+v, want one named Podcasting", ch.Category[0].Sub)
	}
	if ch.Category[1].AttrText != "Society & Culture" {
		t.Errorf("category[1] = %q, want %q", ch.Category[1].AttrText, "Society & Culture")
	}

	if len(ch.Item) != 1 {
		t.Fatalf("got %d items, want 1", len(ch.Item))
	}
	item := ch.Item[0]

	for _, tc := range []struct{ name, got, want string }{
		{"title", item.Title, "Episode One"},
		{"guid", item.Guid.Text, "ep-1-guid"},
		{"guid isPermaLink", item.Guid.IsPermaLink, "false"},
		{"pubDate", item.PubDate, "Wed, 01 Jan 2025 00:00:00 +0000"},
		{"enclosure url", item.Enclosure.URL, "https://example.com/1.mp3"},
		{"enclosure length", item.Enclosure.Length, "12345"},
		{"enclosure type", item.Enclosure.Type, "audio/mpeg"},
		{"duration", item.Duration, "01:02:03"},
		{"explicit", item.Explicit, "false"},
		{"episode number", item.EpisodeNumber, "1"},
		{"season number", item.SeasonNumber, "2"},
		{"episode type", item.EpisodeType, "full"},
		{"episode image", item.Image.Href, "https://example.com/ep1.jpg"},
		{"content:encoded", item.ContentEncoded, "<p>Full show notes</p>"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestParseConvertsCharset(t *testing.T) {
	// "Jos\xe9" is latin-1 for "José"; it is not valid UTF-8, so this only
	// decodes if the CharsetReader is wired up.
	feed := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>" +
		"<rss version=\"2.0\"><channel><title>Jos\xe9</title></channel></rss>"

	rss, err := ParseBytes([]byte(feed))
	if err != nil {
		t.Fatalf("parsing latin-1 feed: %v", err)
	}
	if rss.Channel.Title != "José" {
		t.Errorf("title = %q, want %q", rss.Channel.Title, "José")
	}
}

func TestParseStandardRSSImageFallback(t *testing.T) {
	feed := `<rss version="2.0"><channel>
	  <image><url>https://example.com/plain.png</url></image>
	</channel></rss>`

	rss, err := ParseBytes([]byte(feed))
	if err != nil {
		t.Fatal(err)
	}
	if got := rss.Channel.ImageURL(); got != "https://example.com/plain.png" {
		t.Errorf("ImageURL() = %q, want https://example.com/plain.png", got)
	}
}

func TestParseRejectsMalformedXML(t *testing.T) {
	cases := map[string]string{
		"truncated": `<rss version="2.0"><channel><title>unclosed`,
		"not xml":   `<!DOCTYPE html><html><body>404 Not Found</body></html>`,
		"empty":     ``,
	}
	for name, feed := range cases {
		if _, err := ParseBytes([]byte(feed)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

// Parse runs the decoder non-strict so that the many slightly-invalid feeds in
// the wild still decode. Mismatched end tags are part of what that tolerates,
// so they deliberately do not error.
func TestParseToleratesMismatchedTags(t *testing.T) {
	rss, err := ParseBytes([]byte(`<rss><channel><title>Sloppy</title></rss></channel>`))
	if err != nil {
		t.Fatalf("non-strict parse should tolerate mismatched tags: %v", err)
	}
	if rss.Channel.Title != "Sloppy" {
		t.Errorf("title = %q, want Sloppy", rss.Channel.Title)
	}
}

func TestGetFeedWithContextReturnsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusGone)
	}))
	defer srv.Close()

	_, err := NewRssFeedService().GetFeedWithContext(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected an error for a 410 response")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("error is %T, want *HTTPError", err)
	}
	if httpErr.StatusCode != http.StatusGone {
		t.Errorf("StatusCode = %d, want %d", httpErr.StatusCode, http.StatusGone)
	}
}

func TestGetFeedWithContextHonoursCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := NewRssFeedService().GetFeedWithContext(ctx, srv.URL); err == nil {
		t.Fatal("expected an error from a cancelled context")
	}
}

func TestGetFeedUsesInjectedClient(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(sampleFeed))
	}))
	defer srv.Close()

	fs := NewRssFeedService(WithHTTPClient(srv.Client()))
	rss, err := fs.GetFeedWithContext(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if rss.Channel.Title != "Test Podcast" {
		t.Errorf("title = %q, want Test Podcast", rss.Channel.Title)
	}
	if !strings.Contains(gotAccept, "rss") {
		t.Errorf("Accept header = %q, want it to mention rss", gotAccept)
	}
}
