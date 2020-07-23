// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"
)

type PlayingBar struct {
	*tview.Frame
	full      int
	limit     int
	progress  chan int
	_progress int
	skip      bool
	text      *tview.TextView
}

func (p *PlayingBar) Help() []string {
	return []string{}
}

// Playing bar shows progress of the song and the title of the song
func NewPlayingBar() *PlayingBar {

	textView := tview.NewTextView().SetTextAlign(tview.AlignCenter)

	frame := tview.NewFrame(textView).SetBorders(1, 1, 1, 1, 1, 1)
	frame.SetBorder(true).SetTitle(" Now Playing ")

	p := &PlayingBar{
		Frame:    frame,
		text:     textView,
		progress: make(chan int),
	}

	p.SetDefault()

	textView.SetChangedFunc(func() {
		gomu.App.Draw()

		if !gomu.Player.IsRunning {
			p.SetDefault()
		}
	})

	return p
}

// Start processing progress bar
func (p *PlayingBar) Run() error {

	// When app is suspending, we want the progress bar to stop progressing
	// because it causes screen to hang-up when app starts to stop suspending
	// accumulate when app is suspending
	acc := 0
	wasSuspended := false

	for {

		// stop progressing if song ends or skipped
		if p._progress > p.full || p.skip {
			p.skip = false
			p._progress = 0
			break
		}

		if gomu.IsSuspend {
			// channel the progress to acc
			acc += <-p.progress
			wasSuspended = true
			continue
		} else {
			// normal progressing
			p._progress += <-p.progress
		}

		if wasSuspended {
			// add back so that we dont lose track in progress bar
			p._progress += acc
			wasSuspended = false
			acc = 0
		}

		p.text.Clear()
		start, err := time.ParseDuration(strconv.Itoa(p._progress) + "s")

		if err != nil {
			return tracerr.Wrap(err)
		}

		end, err := time.ParseDuration(strconv.Itoa(p.full) + "s")

		if err != nil {
			return tracerr.Wrap(err)
		}

		x := p._progress * p.limit / p.full
		// our progress bar
		p.text.SetText(fmt.Sprintf("%s ┃%s%s┫ %s",
			fmtDuration(start),
			strings.Repeat("█", x),
			strings.Repeat("━", p.limit-x),
			fmtDuration(end),
		))

	}

	return nil
}

// Updates song title
func (p *PlayingBar) SetSongTitle(title string) {
	p.Clear()
	p.AddText(title, true, tview.AlignCenter, tcell.ColorGreen)
}

// Resets progress bar, ready for execution
func (p *PlayingBar) NewProgress(songTitle string, full, limit int) {
	p.full = full
	p.limit = limit
	p._progress = 0
	p.SetSongTitle(songTitle)
}

// Sets default title and progress bar
func (p *PlayingBar) SetDefault() {
	p.SetSongTitle("---------:---------")
	text := fmt.Sprintf(
		"%s ┣%s┫ %s", "00:00", strings.Repeat("━", 100), "00:00",
	)
	p.text.SetText(text)
}

func (p *PlayingBar) Stop() {
	p.skip = true
}
