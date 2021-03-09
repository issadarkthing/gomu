package lyric

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/asmcos/requests"
	"github.com/ztrue/tracerr"
)

// GetLyricOptionsChinese queries available song lyrics. It returns map of title and
// id of the lyric.
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
	// var results3 []*SongTag
	// for _, v := range results {
	// 	flag := 0
	// 	for _, k := range results {
	// 		if v.TitleForPopup == k.TitleForPopup {
	// 			flag++
	// 		}
	// 	}
	// 	if flag < 2 {
	// 		results3 = append(results3, v)
	// 	}
	// }

	return results, err
}

// GetLyricCn should receive songTag that was returned from GetLyricOptionsCn. GetLyricCn
// returns lyric of the queried song.
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

	var dataMap []map[string]interface{}
	err = resp.Json(&dataMap)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	for _, v := range dataMap {
		songName := v["name"]
		resultName := fmt.Sprintf("%s", songName)
		songArtist := v["artist"]
		resultArtist := fmt.Sprintf("%s", songArtist)
		songAlbum := v["album"]
		resultAlbum := fmt.Sprintf("%s", songAlbum)
		var resultLyricID string
		if serviceProvider == "netease" {
			lyricIDfloat64 := v["lyric_id"]
			resultLyricID = strconv.FormatFloat(lyricIDfloat64.(float64), 'f', -1, 64)
		} else if serviceProvider == "kugou" {
			resultLyricID = v["lyric_id"].(string)
		}
		var resultSongID string
		if serviceProvider == "netease" {
			songIDfloat64 := v["id"]
			resultSongID = strconv.FormatFloat(songIDfloat64.(float64), 'f', -1, 64)
		} else if serviceProvider == "kugou" {
			resultSongID = v["id"].(string)
		}

		resultArtist = strings.TrimPrefix(resultArtist, "[")
		resultArtist = strings.TrimSuffix(resultArtist, "]")
		songTitle := fmt.Sprintf("%s - %s : %s", resultArtist, songName, resultAlbum)
		if resultLyricID == "" || resultSongID == "" {
			continue
		}
		songTag := &SongTag{
			Artist:          resultArtist,
			Title:           resultName,
			Album:           resultAlbum,
			TitleForPopup:   songTitle,
			LangExt:         "zh-CN",
			ServiceProvider: serviceProvider,
			SongID:          resultSongID,
			LyricID:         resultLyricID,
		}
		resultTags = append(resultTags, songTag)
	}

	return resultTags, nil
}
