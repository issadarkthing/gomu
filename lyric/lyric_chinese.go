package lyric

import (
	"fmt"
	// "io"
	// "os"
	"strconv"
	"strings"

	"github.com/asmcos/requests"
	"github.com/ztrue/tracerr"
)

type SongTag struct {
	Artist string
	Title  string
	Album  string
}

// GetLyricOptionsChinese queries available song lyrics. It returns map of title and
// id of the lyric.
func GetLyricOptionsChinese(search string, serviceProvider string) (map[string]string, map[string]SongTag, error) {

	result := make(map[string]string)
	resultTag := make(map[string]SongTag)
	p := requests.Params{
		"site":   serviceProvider,
		"search": search,
	}
	req := requests.Requests()
	req.Header.Set("Content-Type", "application/json")
	resp, err := req.Get("http://api.sunyj.xyz", p)
	if err != nil {
		return nil, nil, tracerr.Wrap(err)
	}

	var dataMap []map[string]interface{}
	err = resp.Json(&dataMap)
	if err != nil {
		return nil, nil, tracerr.Wrap(err)
	}
	for _, v := range dataMap {
		songName := v["name"]
		resultName := fmt.Sprintf("%s", songName)
		songArtist := v["artist"]
		resultArtist := fmt.Sprintf("%s", songArtist)
		songAlbum := v["album"]
		resultAlbum := fmt.Sprintf("%s", songAlbum)
		var lyricID string
		if serviceProvider == "netease" {
			lyricIDfloat64 := v["lyric_id"]
			lyricID = strconv.FormatFloat(lyricIDfloat64.(float64), 'f', -1, 64)
		} else if serviceProvider == "kugou" {
			lyricID = v["lyric_id"].(string)
		}
		resultArtist = strings.TrimPrefix(resultArtist, "[")
		resultArtist = strings.TrimSuffix(resultArtist, "]")
		songTitle := fmt.Sprintf("%s - %s : %s", resultArtist, songName, resultAlbum)
		if lyricID == "" {
			continue
		}
		result[songTitle] = lyricID
		var tag SongTag
		tag.Artist = resultArtist
		tag.Title = resultName
		tag.Album = resultAlbum
		resultTag[lyricID] = tag
	}

	return result, resultTag, nil
}

// GetLyricChinese should receive url that was returned from GetLyricOptions. GetLyric
// returns lyric of the queried song.
func GetLyricChinese(lyricID string, serviceProvider string) (string, error) {

	var lyric string
	p := requests.Params{
		"site":  serviceProvider,
		"lyric": lyricID,
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
		return "", fmt.Errorf("no lyric available")
	}
	// if looksLikeLRC(lyric) {
	// 	// var tmpSubtitle subtitles.Subtitle
	// 	// tmpSubtitle, err = NewFromLRC(lyric)
	// 	// if err != nil {
	// 	// 	return "", err
	// 	// }
	// 	// lyric = tmpSubtitle.AsSRT()
	// 	//Fixme
	// 	filename := "/home/tramhao/old.lrc"
	// 	file, _ := os.Create(filename)
	// 	io.WriteString(file, lyric)
	// 	file.Close()
	// 	var tmpSubtitle Lyric
	// 	tmpSubtitle, err = NewFromLRC(lyric)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	lyric = tmpSubtitle.AsLRC()
	// 	//Fixme
	// 	filename = "/home/tramhao/new.lrc"
	// 	file, _ = os.Create(filename)
	// 	io.WriteString(file, lyric)
	// 	file.Close()
	// }
	if looksLikeLRC(lyric) {
		lyric = cleanLRC(lyric)
		return lyric, nil
	}
	return "", fmt.Errorf("lyric not compatible")
}