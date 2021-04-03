package main

import (
	"sync"

	"github.com/issadarkthing/gomu/player"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"
)

// Command map string to actual command function
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
		if audioFile.isAudioFile {
			return
		}
		err := confirmDeleteAllPopup(audioFile.node)
		if err != nil {
			errorPopup(err)
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
			if len(gomu.queue.items) == 1 && !gomu.player.IsRunning() {
				err := gomu.queue.playQueue()
				if err != nil {
					errorPopup(err)
				}
			}
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
			if len(gomu.queue.items) > 0 && !gomu.player.IsRunning() {
				err := gomu.queue.playQueue()
				if err != nil {
					errorPopup(err)
				}
			}
			return
		}

		confirmationPopup(
			"Are you sure to add this whole directory into queue?",
			func(_ int, label string) {

				if label == "yes" {
					gomu.playlist.addAllToQueue(currNode)
					if len(gomu.queue.items) > 0 && !gomu.player.IsRunning() {
						err := gomu.queue.playQueue()
						if err != nil {
							errorPopup(err)
						}
					}
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

			if gomu.player.IsRunning() {
				gomu.player.Skip()
			} else {
				gomu.queue.playQueue()
			}
		}
	})

	c.define("toggle_loop", func() {
		gomu.queue.isLoop = !gomu.queue.isLoop
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
		gomu.player.TogglePause()
	})

	c.define("volume_up", func() {
		v := player.VolToHuman(gomu.player.GetVolume())
		if v < 100 {
			vol := gomu.player.SetVolume(0.5)
			volumePopup(vol)
		}
	})

	c.define("volume_down", func() {
		v := player.VolToHuman(gomu.player.GetVolume())
		if v > 0 {
			vol := gomu.player.SetVolume(-0.5)
			volumePopup(vol)
		}
	})

	c.define("skip", func() {
		gomu.player.Skip()
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
		if gomu.player.IsRunning() && !gomu.player.IsPaused() {
			position := gomu.playingBar.getProgress() + 10
			if position < gomu.playingBar.getFull() {
				err := gomu.player.Seek(position)
				if err != nil {
					errorPopup(err)
				}
				gomu.playingBar.setProgress(position)
			}
		}
	})

	c.define("rewind", func() {
		if gomu.player.IsRunning() && !gomu.player.IsPaused() {
			position := gomu.playingBar.getProgress() - 10
			if position-1 > 0 {
				err := gomu.player.Seek(position)
				if err != nil {
					errorPopup(err)
				}
				gomu.playingBar.setProgress(position)
			} else {
				err := gomu.player.Seek(0)
				if err != nil {
					errorPopup(err)
				}
				gomu.playingBar.setProgress(0)
			}
		}
	})

	c.define("forward_fast", func() {
		if gomu.player.IsRunning() && !gomu.player.IsPaused() {
			position := gomu.playingBar.getProgress() + 60
			if position < gomu.playingBar.getFull() {
				err := gomu.player.Seek(position)
				if err != nil {
					errorPopup(err)
				}
				gomu.playingBar.setProgress(position)
			}
		}
	})

	c.define("rewind_fast", func() {
		if gomu.player.IsRunning() && !gomu.player.IsPaused() {
			position := gomu.playingBar.getProgress() - 60
			if position-1 > 0 {
				err := gomu.player.Seek(position)
				if err != nil {
					errorPopup(err)
				}
				gomu.playingBar.setProgress(position)
			} else {
				err := gomu.player.Seek(0)
				if err != nil {
					errorPopup(err)
				}
				gomu.playingBar.setProgress(0)
			}
		}
	})

	c.define("yank", func() {
		err := gomu.playlist.yank()
		if err != nil {
			errorPopup(err)
		}
	})

	c.define("paste", func() {
		err := gomu.playlist.paste()
		if err != nil {
			errorPopup(err)
		}
	})

	c.define("repl", func() {
		replPopup()
	})

	c.define("edit_tags", func() {
		audioFile := gomu.playlist.getCurrentFile()
		err := tagPopup(audioFile)
		if err != nil {
			errorPopup(err)
		}
	})

	c.define("switch_lyric", func() {
		gomu.playingBar.switchLyrics()
	})

	c.define("fetch_lyric", func() {
		audioFile := gomu.playlist.getCurrentFile()
		lang := "en"

		var wg sync.WaitGroup
		wg.Add(1)
		if audioFile.isAudioFile {
			go func() {
				err := lyricPopup(lang, audioFile, &wg)
				if err != nil {
					errorPopup(err)
				}
			}()
		}
	})

	c.define("fetch_lyric_cn2", func() {
		audioFile := gomu.playlist.getCurrentFile()
		lang := "zh-CN"

		var wg sync.WaitGroup
		wg.Add(1)
		if audioFile.isAudioFile {
			go func() {
				err := lyricPopup(lang, audioFile, &wg)
				if err != nil {
					errorPopup(err)
				}
			}()
		}
	})

	c.define("lyric_delay_increase", func() {
		err := gomu.playingBar.delayLyric(500)
		if err != nil {
			errorPopup(err)
		}
	})

	c.define("lyric_delay_decrease", func() {
		err := gomu.playingBar.delayLyric(-500)
		if err != nil {
			errorPopup(err)
		}
	})

	c.define("show_colors", func() {
		cp := colorsPopup()
		gomu.pages.AddPage("show-color-popup", center(cp, 95, 40), true, true)
		gomu.popups.push(cp)
	})

	for name, cmd := range c.commands {
		err := gomu.anko.DefineGlobal(name, cmd)
		if err != nil {
			logError(err)
		}
	}

}
