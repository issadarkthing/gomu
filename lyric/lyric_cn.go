package lyric

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ztrue/tracerr"
)

// tagNetease is the tag get from netease
type tagNetease struct {
	Album   string   `json:"album"`
	Artist  []string `json:"artist"`
	ID      int64    `json:"id"`
	LyricID int64    `json:"lyric_id"`
	Name    string   `json:"name"`
	PicID   string   `json:"pic_id"`
	Source  string   `json:"source"`
	URLID   int64    `json:"url_id"`
}

// tagKugou is the tag get from kugou
type tagKugou struct {
	Album   string   `json:"album"`
	Artist  []string `json:"artist"`
	ID      string   `json:"id"`
	LyricID string   `json:"lyric_id"`
	Name    string   `json:"name"`
	PicID   string   `json:"pic_id"`
	Source  string   `json:"source"`
	URLID   string   `json:"url_id"`
}

type tagLyric struct {
	Lyric  string `json:"lyric"`
	Tlyric string `json:"tlyric"`
}

// getLyricOptionsCn queries available song lyrics. It returns slice of SongTag
func getLyricOptionsCn(search string) ([]*SongTag, error) {

	serviceProvider := "netease"
	results, err := getLyricOptionsCnByProvider(search, serviceProvider)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	serviceProvider = "kugou"
	results2, err := getLyricOptionsCnByProvider(search, serviceProvider)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	results = append(results, results2...)

	return results, err
}

// getLyricCn should receive songTag that was returned from getLyricOptionsCn
// and returns lyric of the queried song.
func getLyricCn(songTag *SongTag) (lyricString string, err error) {

	urlSearch := "http://api.sunyj.xyz"

	params := url.Values{}
	params.Add("site", songTag.ServiceProvider)
	params.Add("lyric", songTag.LyricID)
	resp, err := http.Get(urlSearch + "?" + params.Encode())
	if err != nil {
		return "", tracerr.Wrap(err)
	}
	defer resp.Body.Close()

	var tagLyric tagLyric
	err = json.NewDecoder(resp.Body).Decode(&tagLyric)
	if err != nil {
		return "", tracerr.Wrap(err)
	}
	lyricString = tagLyric.Lyric
	if lyricString == "" {
		return "", errors.New("no lyric available")
	}

	if looksLikeLRC(lyricString) {
		lyricString = cleanLRC(lyricString)
		return lyricString, nil
	}
	return "", errors.New("lyric not compatible")
}

// getLyricOptionsCnByProvider do the query by provider
func getLyricOptionsCnByProvider(search string, serviceProvider string) (resultTags []*SongTag, err error) {

	urlSearch := "http://api.sunyj.xyz"

	params := url.Values{}
	params.Add("site", serviceProvider)
	params.Add("search", search)
	resp, err := http.Get(urlSearch + "?" + params.Encode())
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	defer resp.Body.Close()

	switch serviceProvider {
	case "kugou":
		var tagKugou []tagKugou
		err = json.NewDecoder(resp.Body).Decode(&tagKugou)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}

		for _, v := range tagKugou {
			resultArtist := strings.Join(v.Artist, " ")
			songName := v.Name
			resultAlbum := v.Album
			songTitleForPopup := fmt.Sprintf("%s - %s : %s", resultArtist, songName, resultAlbum)
			songTag := &SongTag{
				Artist:          resultArtist,
				Title:           v.Name,
				Album:           v.Album,
				TitleForPopup:   songTitleForPopup,
				LangExt:         "zh-CN",
				ServiceProvider: serviceProvider,
				SongID:          v.ID,
				LyricID:         v.LyricID,
			}
			resultTags = append(resultTags, songTag)
		}

	case "netease":
		var tagNetease []tagNetease
		err = json.NewDecoder(resp.Body).Decode(&tagNetease)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}

		for _, v := range tagNetease {
			resultArtist := strings.Join(v.Artist, " ")
			songName := v.Name
			resultAlbum := v.Album
			songTitleForPopup := fmt.Sprintf("%s - %s : %s", resultArtist, songName, resultAlbum)
			songTag := &SongTag{
				Artist:          resultArtist,
				Title:           v.Name,
				Album:           v.Album,
				URL:             "",
				TitleForPopup:   songTitleForPopup,
				LangExt:         "zh-CN",
				ServiceProvider: serviceProvider,
				SongID:          strconv.FormatInt(v.ID, 10),
				LyricID:         strconv.FormatInt(v.LyricID, 10),
			}
			resultTags = append(resultTags, songTag)
		}

	}

	return resultTags, nil
}
