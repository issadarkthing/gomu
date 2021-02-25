// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bogem/id3v2"
	"github.com/martinlindhe/subtitles"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"
)

type PlayingBar struct {
	*tview.Frame
	full      int
	progress  chan int
	progress  int
	skip      bool
	text      *tview.TextView
	hasTag    bool
	tag       *id3v2.Tag
	subtitle  *subtitles.Subtitle
	subtitles []*gomuSubtitle
	langLyric string
}

type gomuSubtitle struct {
	langExt  string
	subtitle *subtitles.Subtitle
}

func (p *PlayingBar) help() []string {
	return []string{}
}

// Playing bar shows progress of the song and the title of the song
func newPlayingBar() *PlayingBar {

	textView := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	frame := tview.NewFrame(textView).SetBorders(1, 1, 1, 1, 1, 1)
	frame.SetBorder(true).SetTitle(" Now Playing ")

	p := &PlayingBar{
		Frame:    frame,
		text:     textView,
		progress: make(chan int),
	}

	return p
}

// Start processing progress bar
func (p *PlayingBar) run() error {

	for {

		// stop progressing if song ends or skipped
		if p.progress > p.full || p.skip {
			p.skip = false
			p.progress = 0
			break
		}

		p._progress += <-p.progress

		p.text.Clear()
		start, err := time.ParseDuration(strconv.Itoa(p.progress) + "s")

		if err != nil {
			return tracerr.Wrap(err)
		}

		end, err := time.ParseDuration(strconv.Itoa(p.full) + "s")

		if err != nil {
			return tracerr.Wrap(err)
		}

		_, _, width, _ := p.GetInnerRect()
		progressBar := progresStr(p.progress, p.full, width/2, "█", "━")
		// our progress bar
		if p.hasTag && p.subtitles != nil {
			for i := range p.subtitles {
				// First we check if the lyric language prefered is presented
				if strings.Contains(p.langLyric, p.subtitles[i].langExt) {
					p.subtitle = p.subtitles[i].subtitle
					break
				}
			}

			// Secondly we check if english lyric is available
			if p.subtitle == nil {
				for i := range p.subtitles {
					if strings.Contains(p.langLyric, "en") {
						p.subtitle = p.subtitles[i].subtitle
						p.langLyric = "en"
						break
					}
				}
			}

			// Finally we display the first lyric
			if p.subtitle == nil {
				p.subtitle = p.subtitles[0].subtitle
				p.langLyric = p.subtitles[0].langExt
			}

			var lyricText string
			if p.subtitle != nil {
				for i := range p.subtitle.Captions {
					startTime := p.subtitle.Captions[i].Start
					endTime := p.subtitle.Captions[i].End
					currentTime := time.Date(0, 1, 1, 0, 0, p.progress, 0, time.UTC)
					if currentTime.After(startTime.Add(-1*time.Second)) && currentTime.Before(endTime) {
						lyricText = strings.Join(p.subtitle.Captions[i].Text, " ")
						break
					} else {
						lyricText = ""
					}
				}
			}

			p.text.SetText(fmt.Sprintf("%s ┃%s┫ %s\n%v",
				fmtDuration(start),
				progressBar,
				fmtDuration(end),
				lyricText,
			))
		} else {
			p.text.SetText(fmt.Sprintf("%s ┃%s┫ %s",
				fmtDuration(start),
				progressBar,
				fmtDuration(end),
			))
		}
		gomu.app.Draw()
	}

	return nil
}

// Updates song title
func (p *PlayingBar) setSongTitle(title string) {
	p.Clear()
	titleColor := gomu.colors.title
	p.AddText(title, true, tview.AlignCenter, titleColor)
}

// Resets progress bar, ready for execution
func (p *PlayingBar) newProgress(currentSong *AudioFile, full int) {
	p.full = full
	p.progress = 0
	p.setSongTitle(currentSong.name)
	p.hasTag = false
	p.subtitles = nil
	p.tag = nil

	var tag *id3v2.Tag
	var err error
	tag, err = id3v2.Open(currentSong.path, id3v2.Options{Parse: true})
	if tag == nil || err != nil {
		logError(err)
	} else {
		p.hasTag = true
		p.tag = tag

		usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))

		for _, f := range usltFrames {
			uslf, ok := f.(id3v2.UnsynchronisedLyricsFrame)
			if !ok {
				die(errors.New("USLT error!"))
			}
			res, err := subtitles.NewFromSRT(uslf.Lyrics)
			if err != nil {
				logError(err)
			}
			subtitle := &gomuSubtitle{
				langExt:  uslf.ContentDescriptor,
				subtitle: &res,
			}
			p.subtitles = append(p.subtitles, subtitle)
			p.langLyric = gomu.anko.GetString("General.lang_lyric")
			if p.langLyric == "" {
				p.langLyric = "en"
			}
		}
	}
	defer tag.Close()
}

// Sets default title and progress bar
func (p *PlayingBar) setDefault() {
	p.setSongTitle("---------:---------")
	_, _, width, _ := p.GetInnerRect()
	text := fmt.Sprintf(
		"%s ┣%s┫ %s", "00:00", strings.Repeat("━", width/2), "00:00",
	)
	p.text.SetText(text)
}

// Skips the current playing song
func (p *PlayingBar) stop() {
	p.skip = true
}

func (p *PlayingBar) switchLyrics() {

	if len(p.subtitles) == 0 {
		return
	}

	var langIndex int
	for i := range p.subtitles {
		if p.subtitles[i].langExt == p.langLyric {
			langIndex = i + 1
			break
		}
	}

	if langIndex >= len(p.subtitles) {
		langIndex = 0
	}

	p.langLyric = p.subtitles[langIndex].langExt
	defaultTimedPopup(" Success ", p.langLyric+" lyric switched successfully.")

}
