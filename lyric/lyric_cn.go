package lyric

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/asmcos/requests"
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

// getLyricOptionsCn queries available song lyrics. It returns slice of SongTag
func getLyricOptionsCn(search string) ([]*SongTag, error) {

	serviceProvider := "netease"
	results, err := getLyricOptionsCnByProvider(search, serviceProvider)
	if err != nil {
		return nil, err
	}
	serviceProvider = "kugou"
	results2, err := getLyricOptionsCnByProvider(search, serviceProvider)
	if err != nil {
		return nil, err
	}

	results = append(results, results2...)

	return results, err
}

// getLyricCn should receive songTag that was returned from getLyricOptionsCn
// and returns lyric of the queried song.
func getLyricCn(songTag *SongTag) (string, error) {

	var lyric string
	p := requests.Params{
		"site":  songTag.ServiceProvider,
		"lyric": songTag.LyricID,
	}
	req := requests.Requests()
	resp, err := req.Get("http://api.sunyj.xyz", p)
	if err != nil {
		return "", tracerr.Wrap(err)
	}
	var dataMap map[string]interface{}
	err = resp.Json(&dataMap)
	if err != nil {
		return "", tracerr.Wrap(err)
	}
	lyric = dataMap["lyric"].(string)
	if lyric == "" {
		return "", errors.New("no lyric available")
	}

	if looksLikeLRC(lyric) {
		lyric = cleanLRC(lyric)
		return lyric, nil
	}
	return "", errors.New("lyric not compatible")
}

// getLyricOptionsCnByProvider do the query by provider
func getLyricOptionsCnByProvider(search string, serviceProvider string) ([]*SongTag, error) {

	var resultTags []*SongTag
	p := requests.Params{
		"site":   serviceProvider,
		"search": search,
	}
	req := requests.Requests()
	req.Header.Set("Content-Type", "application/json")
	resp, err := req.Get("http://api.sunyj.xyz", p)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	switch serviceProvider {
	case "kugou":
		var tagKugou []tagKugou
		err = json.Unmarshal(resp.Content(), &tagKugou)
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
		err = json.Unmarshal(resp.Content(), &tagNetease)
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
