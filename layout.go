package main

import "github.com/rivo/tview"

func Layout(
	app *tview.Application,
	player *Player,
) *tview.Flex {

	flex := tview.NewFlex().
		AddItem(player.tree, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(player.list, 0, 7, false).
			AddItem(player.playingBar.frame, 0, 1, false), 0, 3, false)

	return flex

}
