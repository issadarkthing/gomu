// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

// Wraps error in a formatted way. 
// function name should be supplied on where the error occured
// newly created err should have a newline
func WrapError(fnName string, err error) error {
	return fmt.Errorf("%s: \n%e", fnName, err)
}

// formats duration to my desired output mm:ss
func fmtDuration(input time.Duration) string {

	val := input.Round(time.Second).String()

	if !strings.Contains(val, "m") {
		val = "0m" + val
	}
	val = strings.ReplaceAll(val, "h", ":")
	val = strings.ReplaceAll(val, "m", ":")
	val = strings.ReplaceAll(val, "s", "")
	var result []string

	for _, v := range strings.Split(val, ":") {

		if len(v) < 2 {
			result = append(result, "0"+v)
		} else {
			result = append(result, v)
		}

	}

	return strings.Join(result, ":")
}

// expands tilde alias to /home/user
func expandTilde(_path string) string {

	if !strings.HasPrefix(_path, "~") {
		return _path
	}

	home, err := os.UserHomeDir()

	if err != nil {
		log.Println(err)
	}

	return path.Join(home, strings.TrimPrefix(_path, "~"))

}

// detects the filetype of file
func GetFileContentType(out *os.File) (string, error) {

	buffer := make([]byte, 512)

	_, err := out.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)

	return strings.SplitAfter(contentType, "/")[1], nil
}

// gets the file name by removing extension and path
func GetName(fn string) string {
	return strings.TrimSuffix(path.Base(fn), ".mp3")
}

// this just parsing the output from the ytdl to get the audio path
// this is used because we need to get the song name
// example ~/path/to/song/song.mp3
func extractFilePath(output []byte, dir string) string {

	regexSearch := fmt.Sprintf(`\[ffmpeg\] Destination: %s\/.*.mp3`,
		escapeBackSlash(dir))

	parseAudioPathOnly := regexp.MustCompile(`\/.*mp3$`)

	re := regexp.MustCompile(regexSearch)

	return string(parseAudioPathOnly.Find(re.Find(output)))

}

func escapeBackSlash(input string) string {
	return strings.ReplaceAll(input, "/", `\/`)
}
