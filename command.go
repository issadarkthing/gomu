package main

import (
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"
)

type Command struct {
	commands map[string]func()
}

func newCommand() Command {
	return Command{
		commands: make(map[string]func()),
	}
}

func (c *Command) define(name string, callBack func()) {
	c.commands[name] = callBack
}

func (c Command) getFn(name string) (func(), error) {
	fn, ok := c.commands[name]
	if !ok {
		return nil, tracerr.New("command not found")
	}
	return fn, nil
}

func (c Command) defineCommands() {

	anko := gomu.anko

	/* Playlist */

	c.define("create_playlist", func() {
		name, _ := gomu.pages.GetFrontPage()
		if name != "mkdir-popup" {
			createPlaylistPopup()
		}
	})

	c.define("delete_playlist", func() {
		audioFile := gomu.playlist.getCurrentFile()
		err := gomu.playlist.deletePlaylist(audioFile)
		if err != nil {
			logError(err)
		}
	})

	c.define("delete_file", func() {
		audioFile := gomu.playlist.getCurrentFile()
		// prevent from deleting a directory
		if !audioFile.isAudioFile {
			return
		}

		err := gomu.playlist.deleteSong(audioFile)
		if err != nil {
			logError(err)
		}
	})

	c.define("youtube_search", func() {
		ytSearchPopup()
	})

	c.define("download_audio", func() {

		audioFile := gomu.playlist.getCurrentFile()
		currNode := gomu.playlist.GetCurrentNode()
		if gomu.pages.HasPage("download-input-popup") {
			gomu.pages.RemovePage("download-input-popup")
			gomu.popups.pop()
			return
		}
		// this ensures it downloads to
		// the correct dir
		if audioFile.isAudioFile {
			downloadMusicPopup(audioFile.parent)
		} else {
			downloadMusicPopup(currNode)
		}
	})

	c.define("add_queue", func() {
		audioFile := gomu.playlist.getCurrentFile()
		currNode := gomu.playlist.GetCurrentNode()
		if audioFile.isAudioFile {
			gomu.queue.enqueue(audioFile)
		} else {
			currNode.SetExpanded(true)
		}
	})

	c.define("close_node", func() {
		audioFile := gomu.playlist.getCurrentFile()
		currNode := gomu.playlist.GetCurrentNode()
		// if closing node with no children
		// close the node's parent
		// remove the color of the node

		if audioFile.isAudioFile {
			parent := audioFile.parent
			gomu.playlist.setHighlight(parent)
			parent.SetExpanded(false)
		}
		currNode.Collapse()
	})

	c.define("bulk_add", func() {
		currNode := gomu.playlist.GetCurrentNode()
		bulkAdd := anko.GetBool("General.confirm_bulk_add")

		if !bulkAdd {
			gomu.playlist.addAllToQueue(currNode)
			return
		}

		confirmationPopup(
			"Are you sure to add this whole directory into queue?",
			func(_ int, label string) {

				if label == "yes" {
					gomu.playlist.addAllToQueue(currNode)
				}

			})
	})

	c.define("refresh", func() {
		gomu.playlist.refresh()
	})

	c.define("rename", func() {
		audioFile := gomu.playlist.getCurrentFile()
		renamePopup(audioFile)
	})

	c.define("playlist_search", func() {

		files := make([]string, len(gomu.playlist.getAudioFiles()))

		for i, file := range gomu.playlist.getAudioFiles() {
			files[i] = file.name
		}

		searchPopup("Search", files, func(text string) {

			audio, err := gomu.playlist.findAudioFile(sha1Hex(text))
			if err != nil {
				logError(err)
			}

			gomu.playlist.setHighlight(audio.node)
			gomu.playlist.refresh()
		})
	})

	c.define("reload_config", func() {
		cfg := expandFilePath(*gomu.args.config)
		err := execConfig(cfg)
		if err != nil {
			errorPopup(err)
		}

		infoPopup("successfully reload config file")
	})

	/* Queue */

	c.define("move_down", func() {
		gomu.queue.next()
	})

	c.define("move_up", func() {
		gomu.queue.prev()
	})

	c.define("delete_item", func() {
		gomu.queue.deleteItem(gomu.queue.GetCurrentItem())
	})

	c.define("clear_queue", func() {
		confirmationPopup("Are you sure to clear the queue?",
			func(_ int, label string) {
				if label == "yes" {
					gomu.queue.clearQueue()
				}
			})
	})

	c.define("play_selected", func() {
		if gomu.queue.GetItemCount() != 0 && gomu.queue.GetCurrentItem() != -1 {
			a, err := gomu.queue.deleteItem(gomu.queue.GetCurrentItem())
			if err != nil {
				logError(err)
			}

			gomu.queue.pushFront(a)
			gomu.player.skip()
		}
	})

	c.define("toggle_loop", func() {
		gomu.queue.isLoop = gomu.player.toggleLoop()
		gomu.queue.updateTitle()
	})

	c.define("shuffle_queue", func() {
		gomu.queue.shuffle()
	})

	c.define("queue_search", func() {

		queue := gomu.queue

		audios := make([]string, 0, len(queue.items))
		for _, file := range queue.items {
			audios = append(audios, file.name)
		}

		searchPopup("Songs", audios, func(selected string) {

			index := 0
			for i, v := range queue.items {
				if v.name == selected {
					index = i
				}
			}

			queue.SetCurrentItem(index)
		})
	})

	/* Global */
	c.define("quit", func() {

		confirmOnExit := anko.GetBool("General.confirm_on_exit")

		if !confirmOnExit {
			err := gomu.quit(gomu.args)
			if err != nil {
				logError(err)
			}
		}
		exitConfirmation(gomu.args)
	})

	c.define("toggle_pause", func() {
		gomu.player.togglePause()
	})

	c.define("volume_up", func() {
		v := volToHuman(gomu.player.volume)
		if v < 100 {
			vol := gomu.player.setVolume(0.5)
			volumePopup(vol)
		}
	})

	c.define("volume_down", func() {
		v := volToHuman(gomu.player.volume)
		if v > 0 {
			vol := gomu.player.setVolume(-0.5)
			volumePopup(vol)
		}
	})

	c.define("skip", func() {
		gomu.player.skip()
	})

	c.define("toggle_help", func() {
		name, _ := gomu.pages.GetFrontPage()

		if name == "help-page" {
			gomu.pages.RemovePage(name)
			gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))
		} else {
			helpPopup(gomu.prevPanel)
		}
	})

	c.define("command_search", func() {

		names := make([]string, 0, len(c.commands))
		for commandName := range c.commands {
			names = append(names, commandName)
		}
		searchPopup("Commands", names, func(selected string) {

			for name, fn := range c.commands {
				if name == selected {
					fn()
				}
			}
		})
	})

	c.define("forward", func() {
		if gomu.player.isRunning && !gomu.player.ctrl.Paused {
			position := gomu.playingBar.progress + 10
			if position < gomu.playingBar.full {
				err := gomu.player.seek(position)
				if err != nil {
					logError(err)
				}
				gomu.playingBar.progress = position
			}
		}
	})

	c.define("rewind", func() {
		if gomu.player.isRunning && !gomu.player.ctrl.Paused {
			position := gomu.playingBar.progress - 10
			if position-1 > 0 {
				err := gomu.player.seek(position)
				if err != nil {
					logError(err)
				}
				gomu.playingBar.progress = position
			} else {
				err := gomu.player.seek(0)
				if err != nil {
					logError(err)
				}
				gomu.playingBar.progress = 0
			}
		}
	})

	c.define("forward_fast", func() {
		if gomu.player.isRunning && !gomu.player.ctrl.Paused {
			position := gomu.playingBar.progress + 60
			if position < gomu.playingBar.full {
				err := gomu.player.seek(position)
				if err != nil {
					logError(err)
				}
				gomu.playingBar.progress = position
			}
		}
	})

	c.define("rewind_fast", func() {
		if gomu.player.isRunning && !gomu.player.ctrl.Paused {
			position := gomu.playingBar.progress - 60
			if position-1 > 0 {
				err := gomu.player.seek(position)
				if err != nil {
					logError(err)
				}
				gomu.playingBar.progress = position
			} else {
				err := gomu.player.seek(0)
				if err != nil {
					logError(err)
				}
				gomu.playingBar.progress = 0
			}
		}
	})

	c.define("yank", func() {
		err := gomu.playlist.yank()
		if err != nil {
			logError(err)
		}
	})

	c.define("paste", func() {
		err := gomu.playlist.paste()
		if err != nil {
			logError(err)
		}
	})

	c.define("repl", func() {
		replPopup()
	})

	c.define("edit_tags", func() {
		audioFile := gomu.playlist.getCurrentFile()
		tagPopup(audioFile)
	})

	c.define("switch_lyric", func() {
		gomu.playingBar.switchLyrics()
	})

	c.define("fetch_lyric", func() {
		audioFile := gomu.playlist.getCurrentFile()

		if audioFile.isAudioFile {
			go func() {
				gomu.app.QueueUpdateDraw(func() {
					err := lyricPopup(audioFile)
					if err != nil {
						errorPopup(err)
						gomu.app.Draw()
					}
				})
			}()
		}
	})

	c.define("fetch_lyric_cn2", func() {
		audioFile := gomu.playlist.getCurrentFile()
		serviceProvider := "netease"
		if audioFile.isAudioFile {
			go func() {
				gomu.app.QueueUpdateDraw(func() {
					err := lyricPopupCN(audioFile, serviceProvider)
					if err != nil {
						errorPopup(err)
						gomu.app.Draw()
					}
				})
			}()
		}
	})
	c.define("fetch_lyric_cn3", func() {
		audioFile := gomu.playlist.getCurrentFile()
		serviceProvider := "kugou"
		if audioFile.isAudioFile {
			go func() {
				gomu.app.QueueUpdateDraw(func() {
					err := lyricPopupCN(audioFile, serviceProvider)
					if err != nil {
						errorPopup(err)
						gomu.app.Draw()
					}
				})
			}()
		}
	})

	c.define("lyric_delay_increase", func() {
		err := gomu.playingBar.delayLyric(500)
		if err != nil {
			logError(err)
		}
	})

	c.define("lyric_delay_decrease", func() {
		err := gomu.playingBar.delayLyric(-500)
		if err != nil {
			logError(err)
		}
	})

	for name, cmd := range c.commands {
		err := gomu.anko.Define(name, cmd)
		if err != nil {
			logError(err)
		}
	}

}
