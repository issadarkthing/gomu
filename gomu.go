package main

import (
	"sync"

	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"
)

var VERSION = "N/A"

var gomu *Gomu

type Gomu struct {
	app        *tview.Application
	playingBar *PlayingBar
	queue      *Queue
	playlist   *Playlist
	player     *Player
	pages      *tview.Pages
	colors     *Colors
	command    Command
	// popups is used to manage focus between popups and panels
	popups    Stack
	prevPanel Panel
	panels    []Panel
	args      Args
	isSuspend bool
	mu        sync.Mutex
}

// Creates new instance of gomu with default values
func newGomu() *Gomu {

	gomu := &Gomu{
		colors:  newColor(),
		command: newCommand(),
	}

	return gomu
}

// Initialize childrens/panels this is seperated from
// constructor function `newGomu` so that we can
// test independently
func (g *Gomu) initPanels(app *tview.Application, args Args) {
	g.app = app
	g.playingBar = newPlayingBar()
	g.queue = newQueue()
	g.playlist = newPlaylist(args)
	g.player = newPlayer()
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

// Safely write the IsSuspend state, IsSuspend is used to indicate if we
// are going to suspend the app. This should be used to widgets or
// texts that keeps rendering continuosly or possible to render when the app
// is going to suspend.
// Returns true if app is not in suspend
func (g *Gomu) suspend() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.isSuspend {
		return false
	}
	g.isSuspend = true
	return true
}

// The opposite of Suspend. Returns true if app is in suspend
func (g *Gomu) unsuspend() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.isSuspend {
		return false
	}
	g.isSuspend = false
	return true
}

// Removes the color of the given panel
func (g *Gomu) setUnfocusPanel(panel Panel) {
	g.prevPanel.SetBorderColor(g.colors.foreground)
	g.prevPanel.SetTitleColor((g.colors.foreground))
}

// Quit the application and do the neccessary clean up
func (g *Gomu) quit(args Args) error {

	if !*args.empty {
		err := gomu.queue.saveQueue()
		if err != nil {
			return tracerr.Wrap(err)
		}
	}

	gomu.app.Stop()

	return nil
}
