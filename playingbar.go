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

func PlayingBar(app *tview.Application, player *Player) *Progress {

	textView := tview.NewTextView()

	progress := InitProgressBar(textView)

	progress.frame.SetInputCapture(func (e *tcell.EventKey) *tcell.EventKey {

		return e
	})

	textView.SetChangedFunc(func() {
		app.Draw()

		if !player.IsRunning {
			progress.SetDefault()
		}
	})

	return progress
}

type Progress struct {
	textView  *tview.TextView
	full      int
	limit     int
	progress  chan int
	frame     *tview.Frame
	_progress int
}

// full is the maximum amount of value can be sent to channel
// limit is the progress bar size
func InitProgressBar(txt *tview.TextView) *Progress {
	p := &Progress{textView: txt}
	p.progress = make(chan int)
	p.textView.SetTextAlign(tview.AlignCenter)

	p.frame = tview.NewFrame(p.textView).SetBorders(1, 1, 1, 1, 1, 1)
	p.frame.SetBorder(true).SetTitle("Now Playing")

	p.SetDefault()

	return p
}

func (p *Progress) Run() {

	go func() { // Simple channel status gauge (progress bar)
		for {
			p._progress += <-p.progress

			p.textView.Clear()

			if p._progress > p.full {
				p._progress = 0
				break
			}

			start, err := time.ParseDuration(strconv.Itoa(p._progress) + "s")

			if err != nil {
				panic(err)
			}

			end, err := time.ParseDuration(strconv.Itoa(p.full) + "s")

			if err != nil {
				panic(err)
			}

			x := p._progress * p.limit / p.full
			p.textView.SetText(fmt.Sprintf("%s %s%s %s",
				start.String(),
				strings.Repeat("■", x),
				strings.Repeat("□", p.limit-x),
				end.String(),
			))

		}
	}()
}

func (p *Progress) SetSongTitle(title string) {

	p.frame.Clear()
	p.frame.AddText(title, true, tview.AlignCenter, tcell.ColorGreen)

}

func (p *Progress) NewProgress(songTitle string, full, limit int) {
	p.full = full
	p.limit = limit
	p._progress = 0
	p.SetSongTitle(songTitle)
}

// sets default title and progress bar
func (p *Progress) SetDefault() {
	p.SetSongTitle("-")
	p.textView.SetText(fmt.Sprintf("%s %s %s", "0s", strings.Repeat("□", 100), "0s"))
}
