// Copyright (C) 2020  Raziman

package main

import "github.com/rivo/tview"

// layout is used to organize the panels
func Layout() *tview.Flex {

	flex := tview.NewFlex().
		AddItem(playlist, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(queue, 0, 7, false).
			AddItem(playingBar, 0, 1, false), 0, 3, false)

	return flex

}
