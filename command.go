package main

import (

	"github.com/rivo/tview"
	"github.com/spf13/viper"
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

	c.define("download_audio", func() {

		audioFile := gomu.playlist.getCurrentFile()
		currNode := gomu.playlist.GetCurrentNode()
		if gomu.pages.HasPage("download-popup") {
			gomu.pages.RemovePage("download-popup")
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

		if !viper.GetBool("general.confirm_bulk_add") {
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

		if viper.GetBool("general.fzf") {
			err := gomu.playlist.fuzzyFind()
			if err != nil {
				logError(err)
			}
			return
		}

		files := make([]string, len(gomu.playlist.getAudioFiles()))

		for i, file := range gomu.playlist.getAudioFiles() {
			files[i] = file.name
		}

		searchPopup(files, func(text string) {

			audio, err := gomu.playlist.findAudioFile(sha1Hex(text))
			if err != nil {
				logError(err)
			}

			gomu.playlist.setHighlight(audio.node)
		})
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
    if (gomu.queue.GetItemCount() != 0 && gomu.queue.GetCurrentItem()!= -1) {
      a, err := gomu.queue.deleteItem(gomu.queue.GetCurrentItem())
      if err != nil {
        logError(err)
      }

      gomu.queue.pushFront(a)
      gomu.player.skip()
    }
	})

	c.define("toggle_loop", func() {

		isLoop := gomu.player.toggleLoop()
		var msg string

		if isLoop {
			msg = "Looping current queue"
      gomu.queue.isLoop = true
		} else {
			msg = "Stopped looping current queue"
      gomu.queue.isLoop = false
		}
    
    gomu.queue.updateTitle()
		defaultTimedPopup(" Loop ", msg)
	})

	c.define("shuffle_queue", func() {
		gomu.queue.shuffle()
	})

	c.define("queue_search", func() {

		queue := gomu.queue

		if viper.GetBool("general.fzf") {
			gomu.suspend()
			if err := queue.fuzzyFind(); err != nil {
				logError(err)
			}
			gomu.unsuspend()
			return
		}

		audios := make([]string, 0, len(queue.items))
		for _, file := range queue.items {
			audios = append(audios, file.name)
		}

		searchPopup(audios, func(selected string) {

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
		if !viper.GetBool("general.confirm_on_exit") {
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
		searchPopup(names, func(selected string) {

			for name, fn := range c.commands {
				if name == selected {
					fn()
				}
			}
		})
	})

  c.define("forward", func() {
    if gomu.player.isRunning && ! gomu.player.ctrl.Paused {
      position := gomu.playingBar._progress + 10
      if (position < gomu.playingBar.full) {
        gomu.player.seek(position)
        gomu.playingBar._progress = position - 1
      }
    }
  })

  c.define("rewind", func() {
    if gomu.player.isRunning && ! gomu.player.ctrl.Paused {
      position := gomu.playingBar._progress - 10
      if (position - 1 > 0 ) {
        gomu.player.seek(position)
        gomu.playingBar._progress = position -1
      } else {
        gomu.player.seek(0)
        gomu.playingBar._progress = 0
      }
    }
  })

  c.define("forward_fast", func() {
    if gomu.player.isRunning && ! gomu.player.ctrl.Paused {
      position := gomu.playingBar._progress + 60
      if (position < gomu.playingBar.full) {
        gomu.player.seek(position)
        gomu.playingBar._progress = position - 1
      }
    }
  })

  c.define("rewind_fast", func() {
    if gomu.player.isRunning && ! gomu.player.ctrl.Paused {
      position := gomu.playingBar._progress - 60
      if (position -1 > 0 ) {
        gomu.player.seek(position)
        gomu.playingBar._progress = position - 1
      } else {
        gomu.player.seek(0)
        gomu.playingBar._progress = 0
      }
    }
  })

}
