// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ztrue/tracerr"
)

// Logs erros to /tmp/gomu.log
func logError(err error) {
	log.Println(tracerr.Sprint(err))
}

// Formats duration to my desired output mm:ss
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

// fmtDurationH returns the formatted duration `x hr x min`
func fmtDurationH(input time.Duration) string {

	re := regexp.MustCompile(`\d+s`)
	val := input.Round(time.Second).String()

	// remove seconds
	result := re.ReplaceAllString(val, "")
	result = strings.Replace(result, "h", " hr ", 1)
	result = strings.Replace(result, "m", " min", 1)

	if result == "" {
		return "0 hr 0 min"
	}

	return result
}

// Expands relative path to absolute path and tilde to /home/(user)
func expandFilePath(path string) string {
	p := expandTilde(path)

	if filepath.IsAbs(p) {
		return p
	}

	p, err := filepath.Abs(p)
	if err != nil {
		panic(err)
	}

	return p
}

// Expands tilde alias to /home/user
func expandTilde(_path string) string {
	if !strings.HasPrefix(_path, "~") {
		return _path
	}

	home, err := os.UserHomeDir()

	if err != nil {
		log.Panicln(tracerr.SprintSource(err))
	}

	return path.Join(home, strings.TrimPrefix(_path, "~"))
}

// Detects the filetype of file
func getFileContentType(out *os.File) (string, error) {

	buffer := make([]byte, 512)

	_, err := out.Read(buffer)
	if err != nil {
		return "", tracerr.Wrap(err)
	}

	contentType := http.DetectContentType(buffer)

	return strings.SplitAfter(contentType, "/")[1], nil
}

// Gets the file name by removing extension and path
func getName(fn string) string {
	return strings.TrimSuffix(path.Base(fn), ".mp3")
}

// This just parsing the output from the ytdl to get the audio path
// This is used because we need to get the song name
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

// progresStr creates a simple progress bar
// example: =====-----
func progresStr(progress, maxProgress, maxLength int,
	fill, empty string) string {

	currLength := maxLength * progress / maxProgress

	return fmt.Sprintf("%s%s",
		strings.Repeat(fill, currLength),
		strings.Repeat(empty, maxLength-currLength),
	)
}

// padHex pad the neccessary 0 to create six hex digit
func padHex(r, g, b int32) string {

	var result strings.Builder

	for _, v := range []int32{r, g, b} {
		hex := fmt.Sprintf("%x", v)

		if len(hex) == 1 {
			result.WriteString(fmt.Sprintf("0%s", hex))
		} else {
			result.WriteString(hex)
		}
	}

	return result.String()
}

func validHexColor(color string) bool {
	reg := regexp.MustCompile(`^#([A-Fa-f0-9]{6})$`)
	return reg.MatchString(color)
}

func contains(needle int, haystack []int) bool {
	for _, i := range haystack {
		if needle == i {
			return true
		}
	}
	return false
}

// appendFile appends to a file, create the file if not exists
func appendFile(path string, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if err != os.ErrNotExist {
			return tracerr.Wrap(err)
		}
		// create the neccessary parent directory
		err = os.MkdirAll(filepath.Dir(expandFilePath(path)), os.ModePerm)
		if err != nil {
			return err
		}
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}
