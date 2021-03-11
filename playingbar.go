// Copyright (C) 2020  Raziman

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bogem/id3v2"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"

	"github.com/issadarkthing/gomu/lyric"
)

type PlayingBar struct {
	*tview.Frame
	full                    int
	update                  chan struct{}
	progress                int
	skip                    bool
	text                    *tview.TextView
	hasTag                  bool
	tag                     *id3v2.Tag
	subtitle                *lyric.Lyric
	subtitles               []*gomuSubtitle
	langLyricCurrentPlaying string
}

type gomuSubtitle struct {
	langExt  string
	subtitle *lyric.Lyric
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
		Frame:  frame,
		text:   textView,
		update: make(chan struct{}),
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

		if gomu.player.IsPaused() {
			time.Sleep(1 * time.Second)
			continue
		}

		p.progress = int(gomu.player.GetPosition().Seconds())

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
		var lyricText string
		if p.subtitle != nil {
			for i := range p.subtitle.Captions {
				startTime := p.subtitle.Captions[i].Start.Add(p.subtitle.Offset)
				var endTime time.Time
				if i < len(p.subtitle.Captions)-1 {
					endTime = p.subtitle.Captions[i+1].Start.Add(p.subtitle.Offset)
				} else {
					// Here we display the last lyric until the end of song
					endTime = time.Date(0, 1, 1, 0, 0, p.full, 0, time.UTC)
				}

				currentTime := time.Date(0, 1, 1, 0, 0, p.progress, 0, time.UTC)
				if currentTime.After(startTime.Add(-1*time.Second)) && currentTime.Before(endTime) {
					lyricText = strings.Join(p.subtitle.Captions[i].Text, " ")
					break
				} else {
					lyricText = ""
				}
			}
		}

		gomu.app.QueueUpdateDraw(func() {
			p.text.SetText(fmt.Sprintf("%s ┃%s┫ %s\n\n%v",
				fmtDuration(start),
				progressBar,
				fmtDuration(end),
				lyricText,
			))
		})

		<-time.After(time.Second)
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

	err := p.loadLyrics(currentSong.path)
	if err != nil {
		errorPopup(err)
	}
	langLyricFromConfig := gomu.anko.GetString("General.lang_lyric")
	if langLyricFromConfig == "" {
		langLyricFromConfig = "en"
	}
	if p.hasTag && p.subtitles != nil {
		// First we check if the lyric language prefered is presented
		for i := range p.subtitles {
			if strings.Contains(langLyricFromConfig, p.subtitles[i].langExt) {
				p.subtitle = p.subtitles[i].subtitle
				p.langLyricCurrentPlaying = p.subtitles[i].langExt
				break
			}
		}

		// Secondly we check if english lyric is available
		if p.subtitle == nil {
			for i := range p.subtitles {
				if p.subtitles[i].langExt == "en" {
					p.subtitle = p.subtitles[i].subtitle
					p.langLyricCurrentPlaying = "en"
					break
				}
			}
		}

		// Finally we display the first lyric
		if p.subtitle == nil {
			p.subtitle = p.subtitles[0].subtitle
			p.langLyricCurrentPlaying = p.subtitles[0].langExt
		}
	}
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

// When switch lyrics, we reload the lyrics from mp3 to reflect changes
func (p *PlayingBar) switchLyrics() {

	err := p.loadLyrics(gomu.player.GetCurrentSong().Path())
	if err != nil {
		errorPopup(err)
	}
	// no subtitle just ignore
	if len(p.subtitles) == 0 {
		return
	}

	// only 1 subtitle, prompt to the user and select this one
	if len(p.subtitles) == 1 {
		defaultTimedPopup(" Warning ", p.langLyricCurrentPlaying+" lyric is the only lyric available")
		p.langLyricCurrentPlaying = p.subtitles[0].langExt
		p.subtitle = p.subtitles[0].subtitle
		return
	}

	// more than 1 subtitle, parse them and select next
	var langIndex int
	for i := range p.subtitles {
		if p.subtitles[i].langExt == p.langLyricCurrentPlaying {
			langIndex = i + 1
			break
		}
	}

	if langIndex >= len(p.subtitles) {
		langIndex = 0
	}

	p.langLyricCurrentPlaying = p.subtitles[langIndex].langExt
	p.subtitle = p.subtitles[langIndex].subtitle

	defaultTimedPopup(" Success ", p.langLyricCurrentPlaying+" lyric switched successfully.")

}

func (p *PlayingBar) delayLyric(lyricDelay int) (err error) {

	p.subtitle.Offset += (time.Duration)(lyricDelay) * time.Millisecond
	err = embedLyric(gomu.player.GetCurrentSong().Path(), p.subtitle.AsLRC(), p.langLyricCurrentPlaying, false)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (p *PlayingBar) loadLyrics(currentSongPath string) error {
	p.hasTag = false
	p.tag = nil
	p.subtitles = nil
	p.subtitle = nil

	var tag *id3v2.Tag
	var err error
	tag, err = id3v2.Open(currentSongPath, id3v2.Options{Parse: true})
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer tag.Close()

	if tag == nil {
		return nil
	}
	p.hasTag = true
	p.tag = tag

	usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))

	for _, f := range usltFrames {
		uslf, ok := f.(id3v2.UnsynchronisedLyricsFrame)
		if !ok {
			return fmt.Errorf("USLT error")
		}
		res, err := lyric.NewFromLRC(uslf.Lyrics)
		if err != nil {
			return tracerr.Wrap(err)
		}
		subtitle := &gomuSubtitle{
			langExt:  uslf.ContentDescriptor,
			subtitle: &res,
		}
		p.subtitles = append(p.subtitles, subtitle)
	}
	return nil
}
