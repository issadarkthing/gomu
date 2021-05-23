// Copyright (C) 2020  Raziman

package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/disintegration/imaging"
	"github.com/rivo/tview"
	"github.com/tramhao/id3v2"
	"github.com/ztrue/tracerr"
	ugo "gitlab.com/diamondburned/ueberzug-go"

	"github.com/issadarkthing/gomu/lyric"
	"github.com/issadarkthing/gomu/player"
)

// PlayingBar shows song name, progress and lyric
type PlayingBar struct {
	*tview.Frame
	full             int32
	update           chan struct{}
	progress         int32
	skip             bool
	text             *tview.TextView
	hasTag           bool
	tag              *id3v2.Tag
	subtitle         *lyric.Lyric
	subtitles        []*lyric.Lyric
	albumPhoto       *ugo.Image
	albumPhotoSource image.Image
	colrowPixel      int32
	mu               sync.RWMutex
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

		p.setProgress(int(gomu.player.GetPosition().Seconds()))

		start, err := time.ParseDuration(strconv.Itoa(progress) + "s")
		if err != nil {
			return tracerr.Wrap(err)
		}

		end, err := time.ParseDuration(strconv.Itoa(full) + "s")

		if err != nil {
			return tracerr.Wrap(err)
		}
		var width, colrowPixel int
		oldColRowPixel := p.getColRowPixel()
		gomu.app.QueueUpdate(func() {

			_, _, width, _ = p.GetInnerRect()

			_, _, _, colPixel, rowPixel, err := p.getConsoleSize()
			if err != nil {
				return
			}

			colrowPixel = rowPixel + colPixel

		})

		progressBar := progresStr(progress, full, width/2, "█", "━")
		if oldColRowPixel != colrowPixel {
			p.updatePhoto()
			p.setColRowPixel(colrowPixel)
		}

		// our progress bar
		var lyricText string
		if p.subtitle != nil {
			lyricText, err = p.subtitle.GetText(progress)
			if err != nil {
				return tracerr.Wrap(err)
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
	p.mu.RLock()
	defer p.mu.RUnlock()
	p.Clear()
	titleColor := gomu.colors.title
	p.AddText(title, true, tview.AlignCenter, titleColor)

}

// Resets progress bar, ready for execution
func (p *PlayingBar) newProgress(currentSong *player.AudioFile, full int) {
	p.setFull(full)
	p.setProgress(0)
	p.hasTag = false
	p.tag = nil
	p.subtitles = nil
	p.mu.RLock()
	p.subtitle = nil
	p.mu.RUnlock()
	if p.albumPhoto != nil {
		p.albumPhoto.Clear()
		p.albumPhoto.Destroy()
		p.albumPhoto = nil
	}

	err := p.loadLyrics(currentSong.Path())
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
				p.mu.RLock()
				p.subtitle = v
				p.mu.RUnlock()
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
	p.setSongTitle(currentSong.Name())

}

// Sets default title and progress bar
func (p *PlayingBar) setDefault() {
	p.setSongTitle("---------:---------")
	_, _, width, _ := p.GetInnerRect()
	text := fmt.Sprintf(
		"%s ┣%s┫ %s", "00:00", strings.Repeat("━", width/2), "00:00",
	)
	p.text.SetText(text)
	if p.albumPhoto != nil {
		p.albumPhoto.Clear()
	}
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
		p.subtitle.Offset -= int32(lyricDelay)
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

	if p.albumPhoto != nil {
		p.albumPhoto.Clear()
		p.albumPhoto.Destroy()
		p.albumPhoto = nil
	}

	syltFrames := tag.GetFrames(tag.CommonID("Synchronised lyrics/text"))
	usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))

	for _, f := range syltFrames {
		sylf, ok := f.(id3v2.SynchronisedLyricsFrame)
		if !ok {
			return errors.New("sylt error")
		}
		for _, u := range usltFrames {
			uslf, ok := u.(id3v2.UnsynchronisedLyricsFrame)
			if !ok {
				return errors.New("uslt error")
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
		imgTmp, err := imaging.Decode(bytes.NewReader(pic.Picture))
		if err != nil {
			return tracerr.Wrap(err)
		}

		p.albumPhotoSource = imgTmp
		p.setColRowPixel(0)
	}

	return nil
}

func (p *PlayingBar) getProgress() int {
	return int(atomic.LoadInt32(&p.progress))
}

func (p *PlayingBar) setProgress(progress int) {
	atomic.StoreInt32(&p.progress, int32(progress))
}

func (p *PlayingBar) getFull() int {
	return int(atomic.LoadInt32(&p.full))
}

func (p *PlayingBar) setFull(full int) {
	atomic.StoreInt32(&p.full, int32(full))
}

func (p *PlayingBar) getColRowPixel() int {
	return int(atomic.LoadInt32(&p.colrowPixel))
}

func (p *PlayingBar) setColRowPixel(colrowPixel int) {
	atomic.StoreInt32(&p.colrowPixel, int32(colrowPixel))
}

// updatePhoto finish two tasks: 1. resize photo based on room left for photo
// 2. register photo in the correct position
func (p *PlayingBar) updatePhoto() {
	// Put the whole block in goroutine, in order not to block the whole apps
	// also to avoid data race by adding QueueUpdateDraw
	go gomu.app.QueueUpdateDraw(func() {
		if p.albumPhotoSource == nil {
			return
		}

		if p.albumPhoto != nil {
			p.albumPhoto.Clear()
			p.albumPhoto.Destroy()
			p.albumPhoto = nil
		}

		// get related size
		x, y, width, colPixel, rowPixel, err := p.getConsoleSize()
		if err != nil {
			logError(err)
			return
		}

		imageWidth := width * colPixel / 3

		// resize the photo according to space left for x and y axis
		dstImage := imaging.Resize(p.albumPhotoSource, imageWidth, 0, imaging.Lanczos)
		positionX := x*colPixel + width*colPixel - dstImage.Rect.Dx() - colPixel
		positionY := y*rowPixel - dstImage.Rect.Dy() - rowPixel/2

		// register new image
		p.albumPhoto, err = ugo.NewImage(dstImage, positionX, positionY)
		if err != nil {
			logError(err)
			return
		}
		p.albumPhoto.Show()

	})
}

func (p *PlayingBar) getConsoleSize() (int, int, int, int, int, error) {
	// get colums and rows count
	x, y, width, height := p.GetRect()

	cols := x + width
	rows := y + height

	// get terminal size
	windowWidth, windowHeight, err := ugo.GetParentSize()
	if err != nil {
		return 0, 0, 0, 0, 0, tracerr.Wrap(err)
	}

	colPixel := windowWidth / cols
	rowPixel := windowHeight / rows
	colrowPixel := rowPixel + colPixel

	p.setColRowPixel(colrowPixel)

	return x, y, width, colPixel, rowPixel, nil
}
