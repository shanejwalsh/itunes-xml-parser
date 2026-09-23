// Package itunes is a client for the iTunes Search API.
package itunes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DefaultBaseURL is the iTunes Search API root.
const DefaultBaseURL = "https://itunes.apple.com"

// HTTPError is returned when the API responds with a non-2xx status. Without
// it a 403 or 503 HTML error page reaches the JSON decoder and surfaces as a
// confusing syntax error.
type HTTPError struct {
	StatusCode int
	Status     string
	URL        string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("itunes: GET %s: %s", e.URL, e.Status)
}

type Result struct {
	WrapperType            string    `json:"wrapperType"`
	Kind                   string    `json:"kind"`
	ArtistID               int       `json:"artistId,omitempty"`
	CollectionID           int       `json:"collectionId"`
	TrackID                int       `json:"trackId"`
	ArtistName             string    `json:"artistName"`
	CollectionName         string    `json:"collectionName"`
	TrackName              string    `json:"trackName"`
	CollectionCensoredName string    `json:"collectionCensoredName"`
	TrackCensoredName      string    `json:"trackCensoredName"`
	ArtistViewURL          string    `json:"artistViewUrl,omitempty"`
	CollectionViewURL      string    `json:"collectionViewUrl"`
	FeedURL                string    `json:"feedUrl"`
	TrackViewURL           string    `json:"trackViewUrl"`
	ArtworkURL30           string    `json:"artworkUrl30"`
	ArtworkURL60           string    `json:"artworkUrl60"`
	ArtworkURL100          string    `json:"artworkUrl100"`
	CollectionPrice        float64   `json:"collectionPrice"`
	TrackPrice             float64   `json:"trackPrice"`
	TrackRentalPrice       int       `json:"trackRentalPrice"`
	CollectionHdPrice      int       `json:"collectionHdPrice"`
	TrackHdPrice           int       `json:"trackHdPrice"`
	TrackHdRentalPrice     int       `json:"trackHdRentalPrice"`
	ReleaseDate            time.Time `json:"releaseDate"`
	CollectionExplicitness string    `json:"collectionExplicitness"`
	TrackExplicitness      string    `json:"trackExplicitness"`
	TrackCount             int       `json:"trackCount"`
	Country                string    `json:"country"`
	Currency               string    `json:"currency"`
	PrimaryGenreName       string    `json:"primaryGenreName"`
	ContentAdvisoryRating  string    `json:"contentAdvisoryRating"`
	ArtworkURL600          string    `json:"artworkUrl600"`
	GenreIds               []string  `json:"genreIds"`
	Genres                 []string  `json:"genres"`
}

type SearchResponse struct {
	ResultCount int      `json:"resultCount"`
	Results     []Result `json:"results"`
}

// Option configures an ItunesApiServices.
type Option func(*ItunesApiServices)

// WithHTTPClient makes the service issue requests with c rather than
// http.DefaultClient, which has no timeout.
func WithHTTPClient(c *http.Client) Option {
	return func(ias *ItunesApiServices) {
		if c != nil {
			ias.client = c
		}
	}
}

// WithBaseURL points the service at a different root, which is mainly useful
// for testing against an httptest server.
func WithBaseURL(baseURL string) Option {
	return func(ias *ItunesApiServices) {
		if baseURL != "" {
			ias.baseURL = baseURL
		}
	}
}

// ItunesApiServices is a client for the iTunes Search API.
type ItunesApiServices struct {
	client  *http.Client
	baseURL string
}

// NewItunesApiServices returns a client configured by opts.
func NewItunesApiServices(opts ...Option) *ItunesApiServices {
	ias := &ItunesApiServices{
		client:  http.DefaultClient,
		baseURL: DefaultBaseURL,
	}
	for _, opt := range opts {
		opt(ias)
	}
	return ias
}

// SearchParams are the query parameters for a search. Zero-valued fields are
// omitted, except Entity which defaults to "podcast".
type SearchParams struct {
	Term string
	// Entity defaults to "podcast".
	Entity string
	// Limit caps the number of results. iTunes accepts 1-200 and defaults to
	// 50. Values outside that range are ignored.
	Limit int
	// Country is a two-letter store code, e.g. "GB". Results are
	// store-specific, so this changes what comes back.
	Country string
}

func (p SearchParams) values() url.Values {
	entity := p.Entity
	if entity == "" {
		entity = "podcast"
	}

	query := url.Values{}
	query.Set("entity", entity)
	query.Set("term", p.Term)
	if p.Limit > 0 && p.Limit <= 200 {
		query.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Country != "" {
		query.Set("country", p.Country)
	}
	return query
}

// SearchWithContext searches the iTunes catalogue.
func (ias *ItunesApiServices) SearchWithContext(ctx context.Context, params SearchParams) (SearchResponse, error) {
	return ias.get(ctx, "/search", params.values())
}

// Search searches for podcasts by term, without a deadline.
// Prefer SearchWithContext.
func (ias *ItunesApiServices) Search(term string) (SearchResponse, error) {
	return ias.SearchWithContext(context.Background(), SearchParams{Term: term})
}

// FindByIdWithContext looks a single item up by its iTunes ID.
func (ias *ItunesApiServices) FindByIdWithContext(ctx context.Context, id int) (SearchResponse, error) {
	return ias.get(ctx, "/lookup", url.Values{"id": {strconv.Itoa(id)}})
}

// FindById looks a single item up by its iTunes ID, without a deadline.
// Prefer FindByIdWithContext.
func (ias *ItunesApiServices) FindById(id int) (SearchResponse, error) {
	return ias.FindByIdWithContext(context.Background(), id)
}

func (ias *ItunesApiServices) get(ctx context.Context, path string, query url.Values) (SearchResponse, error) {
	endpoint, err := url.Parse(ias.baseURL)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("itunes: invalid base URL %q: %w", ias.baseURL, err)
	}
	endpoint.Path += path
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return SearchResponse{}, err
	}
	req.Header.Set("Accept", "application/json")

	res, err := ias.client.Do(req)
	if err != nil {
		return SearchResponse{}, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		// Drain a little so the connection can be reused, then discard.
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4<<10))
		return SearchResponse{}, &HTTPError{StatusCode: res.StatusCode, Status: res.Status, URL: endpoint.String()}
	}

	var searchResponse SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return SearchResponse{}, fmt.Errorf("itunes: decoding response from %s: %w", endpoint.String(), err)
	}
	return searchResponse, nil
}
