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

	"github.com/ztrue/tracerr"
)

// Logs erros to /tmp/gomu.log
func logError(err error) {

	tmpDir := os.TempDir()
	logFile := path.Join(tmpDir, "gomu.log")
	file, e := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if e != nil {
		log.Fatalf("Error opening file %s", logFile)
	}

	defer file.Close()

	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	log.Println(tracerr.SprintSource(err))
}

func debugLog(val ...interface{}) {

	tmpDir := os.TempDir()
	logFile := path.Join(tmpDir, "gomu.log")
	file, e := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if e != nil {
		log.Fatalf("Error opening file %s", logFile)
	}

	defer file.Close()

	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	log.Println(val...)
}

// Wraps error in a formatted way.
func wrapError(fnName string, err error) error {
	return fmt.Errorf("%s: \n%e", fnName, err)
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
	result = strings.Replace(result, "m", " min ", 1)

	return result
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
