// Copyright (C) 2020  Raziman

package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/rivo/tview"
	"github.com/spf13/viper"
)

// initialiaze simple log
func appLog(v ...interface{}) {

	tmpDir := os.TempDir()

	logFile := path.Join(tmpDir, "gomu.log")

	file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		log.Fatalf("Error opening file %s", logFile)
	}

	defer file.Close()

	log.SetOutput(file)
	log.Println(v...)
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
		appLog(err)
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
	return strings.TrimSuffix(path.Base(fn), path.Ext(fn))
}

// download audio from youtube audio and adds the song to the selected playlist
func Ytdl(url string, selPlaylist *tview.TreeNode) {

	// lookup if youtube-dl exists
	_, err := exec.LookPath("youtube-dl")

	if err != nil {
		timedPopup(" Error ", "youtube-dl is not in your $PATH", popupTimeout)
		return
	}

	dir := viper.GetString("music_dir")

	selAudioFile := selPlaylist.GetReference().(*AudioFile)
	selPlaylistName := selAudioFile.Name

	timedPopup(" Ytdl ", "Downloading", time.Second*5)

	// specify the output path for ytdl
	outputDir := fmt.Sprintf(
		"%s/%s/%%(title)s.%%(ext)s",
		dir,
		selPlaylistName)

	args := []string{
		"--extract-audio",
		"--audio-format",
		"mp3",
		"--output",
		outputDir,
		url,
	}

	cmd := exec.Command("youtube-dl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	go func() {

		err := cmd.Run()
		if err != nil {
			timedPopup(" Error ", "Error running youtube-dl", time.Second*5)
			return
		}

		playlistPath := path.Join(expandTilde(dir), selPlaylistName)

		downloadedAudioPath := downloadedFilePath(
			stdout.Bytes(), playlistPath)

		playlist.AddSongToPlaylist(downloadedAudioPath, selPlaylist)

		downloadFinishedMessage := fmt.Sprintf("Finished downloading\n%s", 
			path.Base(downloadedAudioPath))

		timedPopup(
			" Ytdl ",
			downloadFinishedMessage, 
			time.Second*5,
		)

		app.Draw()

	}()

}

// this just parsing the output from the ytdl to get the audio path
// this is used because we need to get the song name
// example ~/path/to/song/song.mp3
func downloadedFilePath(output []byte, dir string) string {

	regexSearch := fmt.Sprintf(`\[ffmpeg\] Destination: %s\/.*.mp3`,
		escapeBackSlash(dir))

	parseAudioPathOnly := regexp.MustCompile(`\/.*mp3$`)

	re := regexp.MustCompile(regexSearch)

	return string(parseAudioPathOnly.Find(re.Find(output)))

}

func escapeBackSlash(input string) string {
	return strings.ReplaceAll(input, "/", `\/`)
}
