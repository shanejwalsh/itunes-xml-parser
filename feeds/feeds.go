
package feeds

import (
	"encoding/xml"
	"net/http"
)

type RssFeedService struct{}

func NewRssFeedService() *RssFeedService {
	return &RssFeedService{}
}

type RSS struct {
	XMLName    xml.Name `xml:"rss"`
	Text       string   `xml:",chardata"`
	Media      string   `xml:"media,attr"`
	Itunes     string   `xml:"itunes,attr"`
	Feedburner string   `xml:"feedburner,attr"`
	Version    string   `xml:"version,attr"`
	Channel    struct {
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
		Generator      string `xml:"generator"`
		TotalResults   struct {
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
		Keywords string `xml:"keywords"`
		Category struct {
			Text     string `xml:",chardata"`
			Scheme   string `xml:"scheme,attr"`
			AttrText string `xml:"text,attr"`
		} `xml:"category"`
		Owner struct {
			Text  string `xml:",chardata"`
			Email string `xml:"email"`
			Name  string `xml:"name"`
		} `xml:"owner"`
		Author   string `xml:"author"`
		Explicit string `xml:"explicit"`
		Image    struct {
			Text string `xml:",chardata"`
			Href string `xml:"href,attr"`
		} `xml:"image"`
		Subtitle string `xml:"subtitle"`
		Summary  string `xml:"summary"`
		Item     []struct {
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
			Thumbnail   struct {
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
			Explicit  string `xml:"explicit"`
			Subtitle  string `xml:"subtitle"`
			Summary   string `xml:"summary"`
			Keywords  string `xml:"keywords"`
			OrigLink  string `xml:"origLink"`
			Enclosure struct {
				Text   string `xml:",chardata"`
				URL    string `xml:"url,attr"`
				Length string `xml:"length,attr"`
				Type   string `xml:"type,attr"`
			} `xml:"enclosure"`
			OrigEnclosureLink string `xml:"origEnclosureLink"`
		} `xml:"item"`
		Credit struct {
			Text string `xml:",chardata"`
			Role string `xml:"role,attr"`
		} `xml:"credit"`
		Rating string `xml:"rating"`
	} `xml:"channel"`
}

func (fs *RssFeedService) GetFeed(feedUrl string) (RSS, error) {

	res, err := http.Get(feedUrl)

	if err != nil {
		return RSS{}, err
	}

	defer res.Body.Close()

	var rss RSS

	err = xml.NewDecoder(res.Body).Decode(&rss)

	return rss, err

}
