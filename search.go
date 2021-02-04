package main

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/spf13/viper"
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

func getSearchResult(search string) ([]YoutubeVideo, error) {

	client := &http.Client{}

	search = url.QueryEscape(search)

	domain := viper.GetString("general.invidious_instance")

	req, err := http.NewRequest("GET", domain+`/api/v1/search?q=`+search, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		var resErr ResponseError

		err = json.NewDecoder(res.Body).Decode(&resErr)
		if err != nil {
			return nil, err
		}

		return nil, &resErr
	}

	yt := []YoutubeVideo{}

	err = json.NewDecoder(res.Body).Decode(&yt)
	if err != nil {
		return nil, err
	}

	return yt, nil
}
