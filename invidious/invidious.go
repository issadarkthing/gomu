package invidious

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/ztrue/tracerr"
)

type Invidious struct {
	// Domain of invidious instance which you get from this list:
	// https://github.com/iv-org/documentation/blob/master/Invidious-Instances.md
	Domain string
}

type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (r *ResponseError) Error() string {
	return r.Message
}

type YoutubeVideo struct {
	Title         string `json:"title"`
	LengthSeconds int    `json:"lengthSeconds"`
	VideoId       string `json:"videoId"`
}

// GetSearchQuery fetches query result from an Invidious instance.
func (i *Invidious) GetSearchQuery(query string) ([]YoutubeVideo, error) {

	query = url.QueryEscape(query)

	targetUrl := i.Domain + `/api/v1/search?q=` + query
	yt := []YoutubeVideo{}

	err := getRequest(targetUrl, &yt)
	if err != nil {
		return nil, err
	}

	return yt, nil
}

// GetSuggestions returns video suggestions based on prefix strings. This is the
// same result as youtube search autocomplete.
func (_ *Invidious) GetSuggestions(prefix string) ([]string, error) {

	query := url.QueryEscape(prefix)
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

// getRequest is a helper function that simplifies GET request and parsing the
// json payload.
func getRequest(url string, v interface{}) error {

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return tracerr.Wrap(err)
	}

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
