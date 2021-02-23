package lyric

import (
	"io/ioutil"
	"testing"

	"github.com/martinlindhe/subtitles"
	"github.com/stretchr/testify/assert"
)

func TestCleanHTML(t *testing.T) {

	clean, err := ioutil.ReadFile("./sample-clean.srt")
	if err != nil {
		t.Error(err)
	}

	unclean, err := ioutil.ReadFile("./sample-unclean.srt")
	if err != nil {
		t.Error(err)
	}

	got := cleanHTML(string(unclean))

	assert.Equal(t, string(clean), got)

	_, err = subtitles.NewFromSRT(got)
	if err != nil {
		t.Error(err)
	}
}
