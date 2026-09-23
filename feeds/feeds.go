// Package feeds fetches and parses podcast RSS feeds.
//
// Fetching and parsing are separable: Parse and ParseBytes accept a body you
// obtained yourself, so callers that need conditional GETs, response size
// limits or body hashing can own the HTTP request and still reuse the parser.
package feeds

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/html/charset"
)

// HTTPError is returned when a feed request completes with a non-2xx status.
// Callers can inspect StatusCode to tell "gone" from "rate limited" from
// "temporarily broken".
type HTTPError struct {
	StatusCode int
	Status     string
	URL        string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("feeds: GET %s: %s", e.URL, e.Status)
}

// Option configures an RssFeedService.
type Option func(*RssFeedService)

// WithHTTPClient makes the service issue requests with c rather than
// http.DefaultClient, which has no timeout.
func WithHTTPClient(c *http.Client) Option {
	return func(fs *RssFeedService) {
		if c != nil {
			fs.client = c
		}
	}
}

// RssFeedService fetches and parses RSS feeds.
type RssFeedService struct {
	client *http.Client
}

// NewRssFeedService returns a service configured by opts.
func NewRssFeedService(opts ...Option) *RssFeedService {
	fs := &RssFeedService{client: http.DefaultClient}
	for _, opt := range opts {
		opt(fs)
	}
	return fs
}

type Episode struct {
	Text     string `xml:",chardata"`
	Title    string `xml:"title"`
	Link     string `xml:"link"`
	Category string `xml:"category"`
	Author   string `xml:"author"`
	PubDate  string `xml:"pubDate"`
	Guid     struct {
		Text        string `xml:",chardata"`
		IsPermaLink string `xml:"isPermaLink,attr"`
	} `xml:"guid"`
	Description string `xml:"description"`
	// ContentEncoded is <content:encoded>, the full HTML show notes. Note this
	// is a different element from Content below, which is <media:content>.
	ContentEncoded string `xml:"encoded"`
	Thumbnail      struct {
		Text   string `xml:",chardata"`
		URL    string `xml:"url,attr"`
		Height string `xml:"height,attr"`
		Width  string `xml:"width,attr"`
	} `xml:"thumbnail"`
	Total struct {
		Text string `xml:",chardata"`
		Thr  string `xml:"thr,attr"`
	} `xml:"total"`
	Content struct {
		Text string `xml:",chardata"`
		URL  string `xml:"url,attr"`
		Type string `xml:"type,attr"`
	} `xml:"content"`
	// Image is <itunes:image href="...">, per-episode artwork.
	Image struct {
		Text string `xml:",chardata"`
		Href string `xml:"href,attr"`
	} `xml:"image"`
	Explicit string `xml:"explicit"`
	Duration string `xml:"duration"`
	// EpisodeNumber, SeasonNumber and EpisodeType are <itunes:episode>,
	// <itunes:season> and <itunes:episodeType>. Kept as strings because feeds
	// in the wild put non-numeric junk in them.
	EpisodeNumber     string    `xml:"episode"`
	SeasonNumber      string    `xml:"season"`
	EpisodeType       string    `xml:"episodeType"`
	Subtitle          string    `xml:"subtitle"`
	Summary           string    `xml:"summary"`
	Keywords          string    `xml:"keywords"`
	OrigLink          string    `xml:"origLink"`
	Enclosure         Enclosure `xml:"enclosure"`
	OrigEnclosureLink string    `xml:"origEnclosureLink"`
}

// Enclosure is the <enclosure> element carrying the audio URL.
type Enclosure struct {
	Text   string `xml:",chardata"`
	URL    string `xml:"url,attr"`
	Length string `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// Category is an <itunes:category>. Feeds may declare several, and each may
// nest a single subcategory.
type Category struct {
	Text     string     `xml:",chardata"`
	Scheme   string     `xml:"scheme,attr"`
	AttrText string     `xml:"text,attr"`
	Sub      []Category `xml:"category"`
}

// Channel is the <channel> element of an RSS feed.
type Channel struct {
	Text  string `xml:",chardata"`
	Title string `xml:"title"`
	Link  struct {
		Text   string `xml:",chardata"`
		Atom10 string `xml:"atom10,attr"`
		Rel    string `xml:"rel,attr"`
		Type   string `xml:"type,attr"`
		Href   string `xml:"href,attr"`
	} `xml:"link"`
	Description struct {
		Text string `xml:",chardata"`
		Type string `xml:"type,attr"`
	} `xml:"description"`
	Language       string `xml:"language"`
	ManagingEditor string `xml:"managingEditor"`
	LastBuildDate  string `xml:"lastBuildDate"`
	PubDate        string `xml:"pubDate"`
	Generator      string `xml:"generator"`
	// NewFeedURL is <itunes:new-feed-url>: the publisher telling aggregators
	// that the feed has permanently moved.
	NewFeedURL   string `xml:"new-feed-url"`
	TotalResults struct {
		Text       string `xml:",chardata"`
		OpenSearch string `xml:"openSearch,attr"`
	} `xml:"totalResults"`
	StartIndex struct {
		Text       string `xml:",chardata"`
		OpenSearch string `xml:"openSearch,attr"`
	} `xml:"startIndex"`
	ItemsPerPage struct {
		Text       string `xml:",chardata"`
		OpenSearch string `xml:"openSearch,attr"`
	} `xml:"itemsPerPage"`
	Info struct {
		Text string `xml:",chardata"`
		URI  string `xml:"uri,attr"`
	} `xml:"info"`
	Copyright string `xml:"copyright"`
	Thumbnail struct {
		Text string `xml:",chardata"`
		URL  string `xml:"url,attr"`
	} `xml:"thumbnail"`
	Keywords string     `xml:"keywords"`
	Category []Category `xml:"category"`
	Owner    struct {
		Text  string `xml:",chardata"`
		Email string `xml:"email"`
		Name  string `xml:"name"`
	} `xml:"owner"`
	Author   string `xml:"author"`
	Explicit string `xml:"explicit"`
	// Type is <itunes:type>: "episodic" or "serial".
	Type  string `xml:"type"`
	Image struct {
		Text string `xml:",chardata"`
		Href string `xml:"href,attr"`
		// URL is the child element of a standard RSS <image><url>...</url>.
		URL string `xml:"url"`
	} `xml:"image"`
	Subtitle string    `xml:"subtitle"`
	Summary  string    `xml:"summary"`
	Item     []Episode `xml:"item"`
	Credit   struct {
		Text string `xml:",chardata"`
		Role string `xml:"role,attr"`
	} `xml:"credit"`
	Rating string `xml:"rating"`
}

type RSS struct {
	XMLName    xml.Name `xml:"rss"`
	Text       string   `xml:",chardata"`
	Media      string   `xml:"media,attr"`
	Itunes     string   `xml:"itunes,attr"`
	Feedburner string   `xml:"feedburner,attr"`
	Version    string   `xml:"version,attr"`
	Channel    Channel  `xml:"channel"`
}

// ImageURL returns the channel artwork, preferring the itunes:image href and
// falling back to a standard RSS <image><url>.
func (c Channel) ImageURL() string {
	if c.Image.Href != "" {
		return c.Image.Href
	}
	return c.Image.URL
}

// Parse decodes an RSS feed from r.
//
// The decoder converts non-UTF-8 documents (ISO-8859-1, windows-1252 and the
// rest) and runs non-strict so that the many slightly-malformed feeds in the
// wild still decode. Genuinely broken XML — unbalanced tags, a truncated
// document — still returns an error.
func Parse(r io.Reader) (RSS, error) {
	decoder := xml.NewDecoder(r)
	decoder.CharsetReader = charset.NewReaderLabel
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity

	var rss RSS
	if err := decoder.Decode(&rss); err != nil {
		return RSS{}, err
	}
	return rss, nil
}

// ParseBytes decodes an RSS feed from b.
func ParseBytes(b []byte) (RSS, error) {
	return Parse(bytes.NewReader(b))
}

// GetFeedWithContext fetches and parses the feed at feedUrl.
func (fs *RssFeedService) GetFeedWithContext(ctx context.Context, feedUrl string) (RSS, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedUrl, nil)
	if err != nil {
		return RSS{}, err
	}
	req.Header.Set("Accept", "application/rss+xml, application/xml;q=0.9, text/xml;q=0.8, */*;q=0.5")

	res, err := fs.client.Do(req)
	if err != nil {
		return RSS{}, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		// Drain a little so the connection can be reused, then discard.
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4<<10))
		return RSS{}, &HTTPError{StatusCode: res.StatusCode, Status: res.Status, URL: feedUrl}
	}

	return Parse(res.Body)
}

// GetFeed fetches and parses the feed at feedUrl without a deadline.
// Prefer GetFeedWithContext.
func (fs *RssFeedService) GetFeed(feedUrl string) (RSS, error) {
	return fs.GetFeedWithContext(context.Background(), feedUrl)
}
