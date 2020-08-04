package main

import (
	"sync"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

const VERSION = "v1.4.1"

var gomu *Gomu

type Gomu struct {
	app         *tview.Application
	playingBar  *PlayingBar
	queue       *Queue
	playlist    *Playlist
	player      *Player
	pages       *tview.Pages
	popups      Stack
	prevPanel   Panel
	popupBg     tcell.Color
	textColor   tcell.Color
	accentColor tcell.Color
	panels      []Panel
	isSuspend   bool
	mu          sync.Mutex
}

// Creates new instance of gomu with default values
func newGomu() *Gomu {

	gomu := &Gomu{
		popupBg:     tcell.GetColor(viper.GetString("color.popup")),
		textColor:   tcell.GetColor(viper.GetString("color.foreground")),
		accentColor: tcell.GetColor(viper.GetString("color.accent")),
	}

	return gomu
}

// Initialize childrens/panels this is seperated from
// constructor function `NewGomu` so that we can
// test independently
func (g *Gomu) initPanels(app *tview.Application) {
	g.app = app
	g.playingBar = newPlayingBar()
	g.queue = newQueue()
	g.playlist = newPlaylist()
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

// Changes title and border color when focusing panel
// and changes color of the previous panel as well
func (g *Gomu) setFocusPanel(panel Panel) {

	g.app.SetFocus(panel.(tview.Primitive))
	panel.SetBorderColor(g.accentColor)
	panel.SetTitleColor(g.accentColor)

	if g.prevPanel == nil {
		return
	}

	g.setUnfocusPanel(g.prevPanel)
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
	g.prevPanel.SetBorderColor(g.textColor)
	g.prevPanel.SetTitleColor((g.textColor))
}

// Quit the application and do the neccessary clean up
func (g *Gomu) quit() error {

	if err := gomu.queue.saveQueue(); err != nil {
		return tracerr.Wrap(err)
	}

	gomu.app.Stop()

	return nil
}
