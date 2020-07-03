// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func InitPlayingBar() *PlayingBar {

	textView := tview.NewTextView().SetTextAlign(tview.AlignCenter)

	progress := InitProgressBar(textView, player)

	textView.SetChangedFunc(func() {
		app.Draw()

		if !player.IsRunning {
			progress.SetDefault()
		}
	})

	return progress
}

type PlayingBar struct {
	*tview.Frame
	full      int
	limit     int
	progress  chan int
	_progress int
	skip      bool
	text      *tview.TextView
}

// full is the maximum amount of value can be sent to channel
// limit is the progress bar size
func InitProgressBar(txt *tview.TextView, player *Player) *PlayingBar {

	frame := tview.NewFrame(txt).SetBorders(1, 1, 1, 1, 1, 1)
	frame.SetBorder(true).SetTitle(" Now Playing ")

	p := &PlayingBar{frame, 0, 0, make(chan int), 0, false, txt}

	p.SetDefault()

	return p
}

func (p *PlayingBar) Run() {

	go func() {
		for {

			if p._progress > p.full || p.skip {
				p.skip = false
				p._progress = 0
				break
			}

			p._progress += <-p.progress

			p.text.Clear()

			start, err := time.ParseDuration(strconv.Itoa(p._progress) + "s")

			if err != nil {
				panic(err)
			}

			end, err := time.ParseDuration(strconv.Itoa(p.full) + "s")

			if err != nil {
				panic(err)
			}

			x := p._progress * p.limit / p.full
			p.text.SetText(fmt.Sprintf("%s ┃%s%s┫ %s",
				fmtDuration(start),
				strings.Repeat("█", x),
				strings.Repeat("━", p.limit-x),
				fmtDuration(end),
			))

		}
	}()
}

func (p *PlayingBar) SetSongTitle(title string) {
	p.Clear()
	p.AddText(title, true, tview.AlignCenter, tcell.ColorGreen)
}

func (p *PlayingBar) NewProgress(songTitle string, full, limit int) {
	p.full = full
	p.limit = limit
	p._progress = 0
	p.SetSongTitle(songTitle)
}

// sets default title and progress bar
func (p *PlayingBar) SetDefault() {
	p.SetSongTitle("---------:---------")
	p.text.SetText(fmt.Sprintf("%s ┣%s┫ %s", "00:00", strings.Repeat("━", 100), "00:00"))
}

func (p *PlayingBar) Stop() {
	p.skip = true
}
