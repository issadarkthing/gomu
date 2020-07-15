package main

import (
	"testing"
)

func Test_NewPlayingBar(t *testing.T) {

	p := NewPlayingBar()

	if p.progress == nil {
		t.Errorf("chan int == nil")
	}

}

func Test_NewProgress(t *testing.T) {

	p := NewPlayingBar()
	full := 100
	limit := 100
	p.NewProgress("sample", full, limit)

	if p.full != full {
		t.Errorf("Expected %d; got %d", full, p.full)	
	}

	if p.limit != limit {
		t.Errorf("Expected %d; got %d", limit, p.limit)	
	}

	if p._progress != 0 {
		t.Errorf("Expected %d; got %d", 0, p._progress)	
	}

}

func Test_Stop(t *testing.T) {

	p := NewPlayingBar()

	p.Stop()

	if p.skip == false {
		t.Errorf("Expected %t; got %t", true, p.skip)
	}
}
