
# Gomu (Go Music Player) [![Go Report Card](https://goreportcard.com/badge/github.com/issadarkthing/gomu)](https://goreportcard.com/report/github.com/issadarkthing/gomu)
Gomu is a Terminal User Interface **TUI** that plays your mp3 files in your local directory. 

![demo](/gomu-demo.gif)

## Features
- lightweight
- simple
- fast
- show audio files as tree
- vim keybindings
- youtube-dl integration
- audio file management

## Installation
```sh
go get -u github.com/issadarkthing/gomu
```

## Configuration
By default, gomu will look for audio files in `~/music` directory. If you wish to change to your desired location, edit `~/.config/gomu/config` file
and change `music_dir: path/to/your/musicDir`. Example of the config file will look like:

```
confirm_on_exit: true
music_dir: ~/music
confirm_bulk_add: true
popup_timeout: 5
```



## Project Background
I just want to implement my own music player with a programming language im currently learning [Go](https://golang.org/). Gomu might not be stable as it in constant development. For now, it can do basic music player can do like adding, deleting songs from queue, skip, play, pause but not seeking or more advanced stuff; feel free to contribute :)
