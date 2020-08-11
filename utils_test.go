package main

import (
	"os"
	"testing"
	"time"
)

func TestFmtDuration(t *testing.T) {

	samples := map[time.Duration]string{
		time.Second * 5:                "00:05",
		time.Hour * 2:                  "02:00:00",
		time.Minute*4 + time.Second*15: "04:15",
		time.Minute * 0:                "00:00",
		time.Millisecond * 5:           "00:00",
	}

	for k, v := range samples {

		got := fmtDuration(k)

		if got != v {
			t.Errorf("fmtDuration(%s); Expected %s got %s", k, v, got)
		}

	}

}

func TestGetName(t *testing.T) {

	samples := map[string]string{
		"hello.mp3":                                "hello",
		"~/music/fl.mp3":                           "fl",
		"/home/terra/Music/pop/hola na.mp3":        "hola na",
		"~/macklemary - (ft jello) extreme!! .mp3": "macklemary - (ft jello) extreme!! ",
	}

	for k, v := range samples {

		got := getName(k)

		if got != v {
			t.Errorf("GetName(%s); Expected %s got %s", k, v, got)
		}
	}
}

func TestDownloadedFilePath(t *testing.T) {

	sample := `[youtube] jJPMnTXl63E: Downloading webpage
[download] Destination: /tmp/Powfu - death bed (coffee for your head) (Official Video) ft. beabadoobee.webm
[download] 100%% of 2.54MiB in 00:0213MiB/s ETA 00:002
[ffmpeg] Destination: /tmp/Powfu - death bed (coffee for your head) (Official Video) ft. beabadoobee.mp3
Deleting original file /tmp/Powfu - death bed (coffee for your head) (Official Video) ft. beabadoobee.webm (pass -k to keep)`

	result := "/tmp/Powfu - death bed (coffee for your head) (Official Video) ft. beabadoobee.mp3"

	got := extractFilePath([]byte(sample), "/tmp")

	if got != result {
		t.Errorf("downloadedFilePath(%s); expected %s got %s", sample, result, got)
	}

}

func TestEscapeBackSlash(t *testing.T) {

	sample := map[string]string{
		"/home/terra":       "\\/home\\/terra",
		"~/Documents/memes": "~\\/Documents\\/memes",
	}

	for k, v := range sample {

		got := escapeBackSlash(k)

		if got != v {
			t.Errorf("escapeBackSlash(%s); expected %s, got %s", k, v, got)
		}
	}
}

func TestExpandTilde(t *testing.T) {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Errorf("Unable to get home dir: %e", err)
	}

	sample := map[string]string{
		"~/music":           homeDir + "/music",
		"/home/terra/Music": homeDir + "/Music",
	}

	for k, v := range sample {

		got := expandTilde(k)

		if got != v {
			t.Errorf("expected %s; got %s", v, got)
		}
	}
}
