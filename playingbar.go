package main

import "github.com/rivo/tview"

func NowPlayingBar() *tview.Box {
	return tview.NewBox().SetBorder(true).
		SetTitle("Currently Playing")
}
