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

	if p.progress == nil {
		t.Errorf("chan int == nil")
	}

}

/* func Test_NewProgress(t *testing.T) {

	p := newPlayingBar()
	full := 100
	p.newProgress("sample", full)

	if p.full != full {
		t.Errorf("Expected %d; got %d", full, p.full)
	}

	if p._progress != 0 {
		t.Errorf("Expected %d; got %d", 0, p._progress)
	}

}
*/
func Test_Stop(t *testing.T) {

	p := newPlayingBar()

	p.stop()

	if p.skip == false {
		t.Errorf("Expected %t; got %t", true, p.skip)
	}
}
