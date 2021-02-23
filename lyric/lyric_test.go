package lyric

import (
	"io/ioutil"
	"testing"

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

	assert.Equal(t, string(clean), cleanHTML(string(unclean)))
}
