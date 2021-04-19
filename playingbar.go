// Copyright (C) 2020  Raziman

package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

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
	width            int32
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
		if p.getWidth() != width {
			p.updatePhoto()
			p.setWidth(width)
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
		p.albumPhoto = nil
	}

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
		imgTmp, err := imaging.Decode(bytes.NewReader(pic.Picture))
		if err != nil {
			return tracerr.Wrap(err)
		}

		p.albumPhotoSource = imgTmp
		p.setWidth(0)
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

func (p *PlayingBar) getWidth() int {
	return int(atomic.LoadInt32(&p.width))
}

func (p *PlayingBar) setWidth(width int) {
	atomic.StoreInt32(&p.width, int32(width))
}

func getConsoleSize() (int, int, int, int) {
	var sz struct {
		rows    uint16
		cols    uint16
		xpixels uint16
		ypixels uint16
	}
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&sz)))
	return int(sz.cols), int(sz.rows), int(sz.xpixels), int(sz.ypixels)
}

func (p *PlayingBar) updatePhoto() {
	go gomu.app.QueueUpdateDraw(func() {
		if p.albumPhotoSource == nil {
			return
		}

		if p.albumPhoto != nil {
			p.albumPhoto.Clear()
			p.albumPhoto.Destroy()
		}
		x, y, width, height := p.GetInnerRect()
		cols, rows, windowWidth, windowHeight := getConsoleSize()

		colPixel := windowWidth / cols
		rowPixel := windowHeight / rows
		remainingX := width/4 - 7
		imageWidth := remainingX*colPixel - colPixel
		if imageWidth > height*rowPixel {
			imageWidth = height * rowPixel
		}
		dstImage := imaging.Fit(p.albumPhotoSource, imageWidth, imageWidth, imaging.Lanczos)
		var err error
		positionX := x*colPixel + remainingX*colPixel/2 - imageWidth/2
		positionY := y*rowPixel + height*rowPixel/2 - imageWidth*10/30
		p.albumPhoto, err = ugo.NewImage(dstImage, positionX, positionY)
		if err != nil {
			errorPopup(err)
		}
		p.albumPhoto.Show()
	})
}
