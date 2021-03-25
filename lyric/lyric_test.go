package lyric

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanHTML(t *testing.T) {

	clean, err := ioutil.ReadFile("./sample-clean.lrc")
	if err != nil {
		t.Error(err)
	}

	unclean, err := ioutil.ReadFile("./sample-unclean.lrc")
	if err != nil {
		t.Error(err)
	}

	got := cleanHTML(string(unclean))

	assert.Equal(t, string(clean), got)

	var lyric Lyric
	err = lyric.NewFromLRC(got)
	if err != nil {
		t.Error(err)
	}
}
