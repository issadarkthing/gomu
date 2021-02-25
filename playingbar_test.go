package main

import (
	"testing"
)

const (
	testConfigPath = "./test/config-test"
)

func Test_NewPlayingBar(t *testing.T) {

	gomu = newGomu()
	err := execConfig(expandFilePath(testConfigPath))
	if err != nil {
		t.Error(err)
	}

	gomu.colors = newColor()

	p := newPlayingBar()

	if p.update == nil {
		t.Errorf("chan int == nil")
	}

}

func Test_NewProgress(t *testing.T) {

	p := newPlayingBar()
	full := 100
	audio := AudioFile{
		path: "./test/rap/audio_test.mp3",
	}

	p.newProgress(&audio, full)

	if p.full != full {
		t.Errorf("Expected %d; got %d", full, p.full)
	}

	if p.progress != 0 {
		t.Errorf("Expected %d; got %d", 0, p.progress)
	}

}

func Test_Stop(t *testing.T) {

	p := newPlayingBar()

	p.stop()

	if p.skip == false {
		t.Errorf("Expected %t; got %t", true, p.skip)
	}
}
