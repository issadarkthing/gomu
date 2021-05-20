package main

import (
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"

	"github.com/issadarkthing/gomu/anko"
	"github.com/issadarkthing/gomu/hook"
	"github.com/issadarkthing/gomu/player"
)

// VERSION is version information of gomu
var VERSION = "N/A"

var gomu *Gomu

// Gomu is the application from tview
type Gomu struct {
	app        *tview.Application
	playingBar *PlayingBar
	queue      *Queue
	playlist   *Playlist
	player     player.Player
	pages      *tview.Pages
	colors     *Colors
	command    Command
	// popups is used to manage focus between popups and panels
	popups    Stack
	prevPanel Panel
	panels    []Panel
	args      Args
	anko      *anko.Anko
	hook      *hook.EventHook
}

// Creates new instance of gomu with default values
func newGomu() *Gomu {

	gomu := &Gomu{
		command: newCommand(),
		anko:    anko.NewAnko(),
		hook:    hook.NewEventHook(),
	}

	return gomu
}

// Initialize childrens/panels this is seperated from
// constructor function `newGomu` so that we can
// test independently
func (g *Gomu) initPanels(app *tview.Application, args Args) {
	var err error
	g.app = app
	g.playingBar = newPlayingBar()
	g.queue = newQueue()
	g.playlist = newPlaylist(args)
	g.player, err = player.NewPlayer(g.anko.GetInt("General.volume"), g.anko.GetString("General.backend_server"), g.anko.GetString("General.mpd_port"))
	if err != nil {
		logError(err)
	}

	g.pages = tview.NewPages()
	g.panels = []Panel{g.playlist, g.queue, g.playingBar}
}

// Cycle between panels
func (g *Gomu) cyclePanels() Panel {

	var anyChildHasFocus bool

	for i, child := range g.panels {

		if child.HasFocus() {

			anyChildHasFocus = true

			var nextChild Panel

			// if its the last element set the child back to one
			if i == len(g.panels)-1 {
				nextChild = g.panels[0]
			} else {
				nextChild = g.panels[i+1]
			}

			g.setFocusPanel(nextChild)

			g.prevPanel = nextChild
			return nextChild
		}
	}

	first := g.panels[0]

	if !anyChildHasFocus {
		g.setFocusPanel(first)
	}

	g.prevPanel = first
	return first
}

func (g *Gomu) cyclePanels2() Panel {
	first := g.panels[0]
	second := g.panels[1]
	if first.HasFocus() {
		g.setFocusPanel(second)
		g.prevPanel = second
		return second
	} else if second.HasFocus() {
		g.setFocusPanel(first)
		g.prevPanel = first
		return first
	} else {
		g.setFocusPanel(first)
		g.prevPanel = first
		return first
	}
}

// Changes title and border color when focusing panel
// and changes color of the previous panel as well
func (g *Gomu) setFocusPanel(panel Panel) {

	g.app.SetFocus(panel.(tview.Primitive))
	panel.SetBorderColor(g.colors.accent)
	panel.SetTitleColor(g.colors.accent)

	if g.prevPanel == nil {
		return
	}

	if g.prevPanel != panel {
		g.setUnfocusPanel(g.prevPanel)
	}
}

// Removes the color of the given panel
func (g *Gomu) setUnfocusPanel(panel Panel) {
	g.prevPanel.SetBorderColor(g.colors.foreground)
	g.prevPanel.SetTitleColor(g.colors.foreground)
}

// Quit the application and do the neccessary clean up
func (g *Gomu) quit(args Args) error {

	if !*args.empty {
		err := gomu.queue.saveQueue()
		if err != nil {
			return tracerr.Wrap(err)
		}
	}

	err := gomu.player.Stop()
	if err != nil {
		return tracerr.Wrap(err)
	}

	if gomu.playingBar.albumPhoto != nil {
		gomu.playingBar.albumPhoto.Destroy()
	}

	gomu.app.Stop()

	return nil
}
