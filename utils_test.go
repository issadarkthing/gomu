package main

import (
	"os"
	"testing"
	"time"

	"github.com/bogem/id3v2"
	"github.com/stretchr/testify/assert"
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
		"~/music":          homeDir + "/music",
		homeDir + "/Music": homeDir + "/Music",
	}

	for k, v := range sample {

		got := expandTilde(k)

		if got != v {
			t.Errorf("expected %s; got %s", v, got)
		}
	}
}

func TestEmbedLyric(t *testing.T) {

	testFile := "./test/sample"
	lyric := "sample"
	descriptor := "en"

	f, err := os.Create(testFile)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	defer func(){
		err := os.Remove(testFile)
		if err != nil {
			t.Error(err)
		}
	}()

	err = embedLyric(testFile, lyric, descriptor)
	if err != nil {
		t.Error(err)
	}

	tag, err := id3v2.Open(testFile, id3v2.Options{Parse: true})
	if err != nil {
		t.Error(err)
	} else if tag == nil {
		t.Error("unable to read tag")
	}

	usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))
	frame, ok := usltFrames[0].(id3v2.UnsynchronisedLyricsFrame)
	if !ok {
		t.Error("invalid type")
	}

	assert.Equal(t, lyric, frame.Lyrics)
	assert.Equal(t, descriptor, frame.ContentDescriptor)
}
