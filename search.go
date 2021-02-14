package main

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/ztrue/tracerr"
)

type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (r *ResponseError) Error() string {
	return r.Message
}

type Thumbnail struct {
	Quality string `json:"quality,omitempty"`
	Url     string `json:"url,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
}

type YoutubeVideo struct {
	Type            string      `json:"type,omitempty"`
	Title           string      `json:"title,omitempty"`
	VideoId         string      `json:"videoId,omitempty"`
	Author          string      `json:"author,omitempty"`
	AuthorId        string      `json:"authorId,omitempty"`
	AuthorUrl       string      `json:"authorUrl,omitempty"`
	VideoThumbnails []Thumbnail `json:"videoThumbnails,omitempty"`
	Description     string      `json:"description,omitempty"`
	DescriptionHtml string      `json:"descriptionHtml,omitempty"`
	ViewCount       int         `json:"viewCount,omitempty"`
	Published       int         `json:"published,omitempty"`
	PublishedText   string      `json:"publishedText,omitempty"`
	LengthSeconds   int         `json:"lengthSeconds,omitempty"`
	LiveNow         bool        `json:"liveNow,omitempty"`
	Paid            bool        `json:"paid,omitempty"`
	Premium         bool        `json:"premium,omitempty"`
	IsUpcoming      bool        `json:"isUpcoming,omitempty"`
}

const userAgent = "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0"

func getRequest(url string, v interface{}) error {

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return tracerr.Wrap(err)
	}

	req.Header.Set("User-Agent", userAgent)

	res, err := client.Do(req)
	if err != nil {
		return tracerr.Wrap(err)
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		var resErr ResponseError

		err = json.NewDecoder(res.Body).Decode(&resErr)
		if err != nil {
			return tracerr.Wrap(err)
		}

		return &resErr
	}

	err = json.NewDecoder(res.Body).Decode(&v)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

func getSearchResult(query string) ([]YoutubeVideo, error) {

	query = url.QueryEscape(query)
	domain := gomu.anko.getString("invidious_instance")

	targetUrl := domain + `/api/v1/search?q=` + query
	yt := []YoutubeVideo{}

	err := getRequest(targetUrl, &yt)
	if err != nil {
		return nil, err
	}

	return yt, nil
}

func getSuggestions(query string) ([]string, error) {

	query = url.QueryEscape(query)
	targetUrl :=
		`http://suggestqueries.google.com/complete/search?client=firefox&ds=yt&q=` + query

	res := []json.RawMessage{}
	err := getRequest(targetUrl, &res)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	suggestions := []string{}
	err = json.Unmarshal(res[1], &suggestions)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	return suggestions, nil
}
