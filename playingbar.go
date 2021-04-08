// Copyright (C) 2020  Raziman

package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/disintegration/imaging"
	"github.com/rivo/tview"
	"github.com/tramhao/id3v2"
	"github.com/ztrue/tracerr"
	ugo "gitlab.com/diamondburned/ueberzug-go"

	"github.com/issadarkthing/gomu/lyric"
)

// PlayingBar shows song name, progress and lyric
type PlayingBar struct {
	*tview.Frame
	full       int64
	update     chan struct{}
	progress   int64
	skip       bool
	text       *tview.TextView
	hasTag     bool
	tag        *id3v2.Tag
	subtitle   *lyric.Lyric
	subtitles  []*lyric.Lyric
	albumPhoto *ugo.Image
}

func (p *PlayingBar) help() []string {
	return []string{}
}

// Playing bar shows progress of the song and the title of the song
func newPlayingBar() *PlayingBar {

	textView := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	textView.SetBackgroundColor(gomu.colors.background)
	textView.SetDynamicColors(true)

	frame := tview.NewFrame(textView).SetBorders(1, 1, 1, 1, 1, 1)
	frame.SetBorder(true).SetTitle(" Now Playing ")
	frame.SetBackgroundColor(gomu.colors.background)

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
		progress := p.getProgress()
		full := p.getFull()

		if progress > full || p.skip {
			p.skip = false
			p.setProgress(0)
			break
		}

		if gomu.player.IsPaused() {
			time.Sleep(1 * time.Second)
			continue
		}

		// p.progress = int(gomu.player.GetPosition().Seconds())
		p.setProgress(int(gomu.player.GetPosition().Seconds()))

		start, err := time.ParseDuration(strconv.Itoa(progress) + "s")
		if err != nil {
			return tracerr.Wrap(err)
		}

		end, err := time.ParseDuration(strconv.Itoa(full) + "s")

		if err != nil {
			return tracerr.Wrap(err)
		}
		var width int
		gomu.app.QueueUpdate(func() {
			_, _, width, _ = p.GetInnerRect()
		})

		progressBar := progresStr(progress, full, width/2, "█", "━")
		// our progress bar
		var lyricText string
		if p.subtitle != nil {
			for i := range p.subtitle.SyncedCaptions {
				startTime := int32(p.subtitle.SyncedCaptions[i].Timestamp)
				var endTime int32
				if i < len(p.subtitle.SyncedCaptions)-1 {
					endTime = int32(p.subtitle.SyncedCaptions[i+1].Timestamp)
				} else {
					// Here we display the last lyric until the end of song
					endTime = int32(full * 1000)
				}

				// here the currentTime is delayed 1 second because we want to show lyrics earlier
				currentTime := int32(progress*1000) + 1000
				if currentTime >= startTime && currentTime <= endTime {
					lyricText = p.subtitle.SyncedCaptions[i].Text
					break
				} else {
					lyricText = ""
				}
			}
		}

		gomu.app.QueueUpdateDraw(func() {
			p.text.SetText(fmt.Sprintf("%s ┃%s┫ %s\n\n[%s]%v[-]",
				fmtDuration(start),
				progressBar,
				fmtDuration(end),
				gomu.colors.subtitle,
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
	p.setFull(full)
	p.setProgress(0)
	p.hasTag = false
	p.tag = nil
	p.subtitles = nil
	p.subtitle = nil
	if p.albumPhoto != nil {
		p.albumPhoto.Clear()
		p.albumPhoto.Destroy()
		ugo.Close()
	}
	p.albumPhoto = nil

	err := p.loadLyrics(currentSong.path)
	if err != nil {
		errorPopup(err)
		return
	}
	langLyricFromConfig := gomu.anko.GetString("General.lang_lyric")
	if langLyricFromConfig == "" {
		langLyricFromConfig = "en"
	}
	if p.hasTag && p.subtitles != nil {
		// First we check if the lyric language preferred is presented
		for _, v := range p.subtitles {
			if strings.Contains(langLyricFromConfig, v.LangExt) {
				p.subtitle = v
				break
			}
		}

		// Secondly we check if english lyric is available
		if p.subtitle == nil {
			for _, v := range p.subtitles {
				if v.LangExt == "en" {
					p.subtitle = v
					break
				}
			}
		}

		// Finally we display the first lyric
		if p.subtitle == nil {
			p.subtitle = p.subtitles[0]
		}
	}
	p.setSongTitle(currentSong.name)

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
		return
	}
	// no subtitle just ignore
	if len(p.subtitles) == 0 {
		defaultTimedPopup(" Warning ", "No embed lyric found")
		p.subtitle = nil
		return
	}

	// only 1 subtitle, prompt to the user and select this one
	if len(p.subtitles) == 1 {
		p.subtitle = p.subtitles[0]
		defaultTimedPopup(" Warning ", p.subtitle.LangExt+" lyric is the only lyric available")
		return
	}

	// more than 1 subtitle, cycle through them and select next
	var langIndex int
	for i, v := range p.subtitles {
		if p.subtitle.LangExt == v.LangExt {
			langIndex = i + 1
			break
		}
	}

	if langIndex >= len(p.subtitles) {
		langIndex = 0
	}

	p.subtitle = p.subtitles[langIndex]

	defaultTimedPopup(" Success ", p.subtitle.LangExt+" lyric switched successfully.")
}

func (p *PlayingBar) delayLyric(lyricDelay int) (err error) {

	if p.subtitle != nil {
		p.subtitle.Offset += int32(lyricDelay)
		err = embedLyric(gomu.player.GetCurrentSong().Path(), p.subtitle, false)
		if err != nil {
			return tracerr.Wrap(err)
		}
		err = p.loadLyrics(gomu.player.GetCurrentSong().Path())
		if err != nil {
			return tracerr.Wrap(err)
		}
		for _, v := range p.subtitles {
			if strings.Contains(v.LangExt, p.subtitle.LangExt) {
				p.subtitle = v
				break
			}
		}
	}
	return nil
}

func (p *PlayingBar) loadLyrics(currentSongPath string) error {
	p.subtitles = nil

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

	syltFrames := tag.GetFrames(tag.CommonID("Synchronised lyrics/text"))
	usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))

	for _, f := range syltFrames {
		sylf, ok := f.(id3v2.SynchronisedLyricsFrame)
		if !ok {
			return fmt.Errorf("sylt error")
		}
		for _, u := range usltFrames {
			uslf, ok := u.(id3v2.UnsynchronisedLyricsFrame)
			if !ok {
				return errors.New("USLT error")
			}
			if sylf.ContentDescriptor == uslf.ContentDescriptor {
				var lyric lyric.Lyric
				err := lyric.NewFromLRC(uslf.Lyrics)
				if err != nil {
					return tracerr.Wrap(err)
				}
				lyric.SyncedCaptions = sylf.SynchronizedTexts
				lyric.LangExt = sylf.ContentDescriptor
				p.subtitles = append(p.subtitles, &lyric)
			}
		}
	}

	pictures := tag.GetFrames(tag.CommonID("Attached picture"))
	for _, f := range pictures {
		pic, ok := f.(id3v2.PictureFrame)
		if !ok {
			return errors.New("picture frame error")
		}

		// Do something with picture frame.
		img1, err := imaging.Decode(bytes.NewReader(pic.Picture))
		if err != nil {
			return tracerr.Wrap(err)
		}
		dstImage128 := imaging.Fit(img1, 128, 128, imaging.Lanczos)

		go gomu.app.QueueUpdateDraw(func() {
			x, y, _, _ := p.GetInnerRect()
			_, err := ugo.NewImage(dstImage128, x*16, y*31)
			if err != nil {
				log.Fatal(err)
			}
		})
	}

	return nil
}

func (p *PlayingBar) getProgress() int {
	return int(atomic.LoadInt64(&p.progress))
}

func (p *PlayingBar) setProgress(progress int) {
	atomic.StoreInt64(&p.progress, int64(progress))
}

func (p *PlayingBar) getFull() int {
	return int(atomic.LoadInt64(&p.full))
}

func (p *PlayingBar) setFull(full int) {
	atomic.StoreInt64(&p.full, int64(full))
}
