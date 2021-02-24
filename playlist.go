// Copyright (C) 2020  Raziman

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bogem/id3v2"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	spin "github.com/tj/go-spin"
	"github.com/ztrue/tracerr"
)

// AudioFile is representing directories and mp3 files
// if isAudioFile equals to false it is a directory
type AudioFile struct {
	name        string
	path        string
	isAudioFile bool
	length      time.Duration
	node        *tview.TreeNode
	parent      *tview.TreeNode
}

// Playlist struct represents playlist panel
// that shows the tree of the music directory
type Playlist struct {
	*tview.TreeView
	prevNode     *tview.TreeNode
	defaultTitle string
	// number of downloads
	download int
	done     chan struct{}
}

var (
	yankFile *AudioFile
	isYanked bool
)

func (p *Playlist) help() []string {

	return []string{
		"j      down",
		"k      up",
		"h      close node",
		"a      create a playlist",
		"l      add song to queue",
		"L      add playlist to queue",
		"d      delete file from filesystem",
		"D      delete playlist from filesystem",
		"Y      download audio from url",
		"r      refresh",
		"R      rename",
		"y/p    yank/paste file",
		"/      find in playlist",
		"s      search audio from youtube",
		"t      edit mp3 tags",
		"1      find lyric if available",
	}

}

// newPlaylist returns new instance of playlist and runs populate function
// on root music directory.
func newPlaylist(args Args) *Playlist {

	anko := gomu.anko

	m := anko.GetString("General.music_dir")
	rootDir, err := filepath.Abs(expandTilde(m))
	if err != nil {
		err = tracerr.Errorf("unable to find music directory: %e", err)
		die(err)
	}

	// if not default value was given
	if *args.music != "~/music" {
		rootDir = expandFilePath(*args.music)
	}

	var rootTextView string

	useEmoji := anko.GetBool("General.use_emoji")

	if useEmoji {

		emojiPlaylist := anko.GetString("Emoji.playlist")
		rootTextView = fmt.Sprintf("%s %s", emojiPlaylist, path.Base(rootDir))

	} else {
		rootTextView = path.Base(rootDir)
	}

	root := tview.NewTreeNode(rootTextView).
		SetColor(gomu.colors.accent)

	tree := tview.NewTreeView().SetRoot(root)

	playlist := &Playlist{
		TreeView:     tree,
		defaultTitle: "─ Playlist ──┤ 0 downloads ├",
		done:         make(chan struct{}),
	}

	rootAudioFile := &AudioFile{
		name: path.Base(rootDir),
		node: root,
		path: rootDir,
	}

	root.SetReference(rootAudioFile)

	playlist.
		SetTitle(playlist.defaultTitle).
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1)

	populate(root, rootDir)

	var firstChild *tview.TreeNode

	if len(root.GetChildren()) == 0 {
		firstChild = root
	} else {
		firstChild = root.GetChildren()[0]
	}

	playlist.setHighlight(firstChild)

	playlist.SetChangedFunc(func(node *tview.TreeNode) {
		playlist.setHighlight(node)
	})

	playlist.SetSelectedFunc(func(node *tview.TreeNode) {
		node.SetExpanded(!node.IsExpanded())
	})

	playlist.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		if gomu.anko.KeybindExists("playlist", e) {

			err := gomu.anko.ExecKeybind("playlist", e)
			if err != nil {
				errorPopup(err)
			}

			return nil
		}

		cmds := map[rune]string{
			'a': "create_playlist",
			'D': "delete_playlist",
			'd': "delete_file",
			'Y': "download_audio",
			's': "youtube_search",
			'l': "add_queue",
			'L': "bulk_add",
			'h': "close_node",
			'r': "refresh",
			'R': "rename",
			'y': "yank",
			'p': "paste",
			'/': "playlist_search",
			't': "edit_tags",
			'1': "fetch_lyric",
		}

		for key, cmd := range cmds {
			if e.Rune() != key {
				continue
			}
			fn, err := gomu.command.getFn(cmd)
			if err != nil {
				errorPopup(err)
				return e
			}
			fn()
		}

		// disable default key handler for space
		if e.Rune() == ' ' {
			return nil
		}

		return e
	})

	return playlist

}

// Returns the current file highlighted in the playlist
func (p Playlist) getCurrentFile() *AudioFile {
	node := p.GetCurrentNode()
	if node == nil {
		return nil
	}
	return node.GetReference().(*AudioFile)
}

// Deletes song from filesystem
func (p *Playlist) deleteSong(audioFile *AudioFile) (err error) {

	confirmationPopup(
		"Are you sure to delete this audio file?", func(_ int, buttonName string) {

			if buttonName == "no" || buttonName == "" {
				return
			}

			audioName := getName(audioFile.path)
			err := os.Remove(audioFile.path)

			if err != nil {

				defaultTimedPopup(" Error ", "Unable to delete "+audioFile.name)

				err = tracerr.Wrap(err)

			} else {

				defaultTimedPopup(" Success ",
					audioFile.name+"\nhas been deleted successfully")
				p.refresh()

				// Here we remove the song from queue
				songPaths := gomu.queue.getItems()
				if audioName == getName(gomu.player.currentSong.name) {
					gomu.player.skip()
				}
				for i, songPath := range songPaths {
					if strings.Contains(songPath, audioName) {
						gomu.queue.deleteItem(i)
					}
				}
			}

		})

	return nil
}

// Deletes playlist/dir from filesystem
func (p *Playlist) deletePlaylist(audioFile *AudioFile) (err error) {

	var selectedDir *AudioFile

	// gets the parent dir if current focused node is not a dir
	if audioFile.isAudioFile {
		selectedDir = audioFile.parent.GetReference().(*AudioFile)
	} else {
		selectedDir = audioFile
	}

	confirmationPopup("Are you sure to delete this directory?",
		func(_ int, buttonName string) {

			if buttonName == "no" || buttonName == "" {
				return
			}

			err := os.RemoveAll(selectedDir.path)

			if err != nil {

				defaultTimedPopup(
					" Error ",
					"Unable to delete dir "+selectedDir.name)

				err = tracerr.Wrap(err)

			} else {

				defaultTimedPopup(
					" Success ",
					selectedDir.name+"\nhas been deleted successfully")

				p.refresh()
			}

		})

	return nil
}

// Bulk add a playlist to queue
func (p *Playlist) addAllToQueue(root *tview.TreeNode) {

	var childrens []*tview.TreeNode
	childrens = root.GetChildren()

	// gets the parent if the highlighted item is a file
	if root.GetReference().(*AudioFile).isAudioFile {
		childrens = root.GetReference().(*AudioFile).parent.GetChildren()
	}

	for _, v := range childrens {
		currNode := v.GetReference().(*AudioFile)
		if currNode.isAudioFile {
			if currNode != gomu.player.currentSong {
				gomu.queue.enqueue(currNode)
			}
		}
	}

}

// Refreshes the playlist and read the whole root music dir
func (p *Playlist) refresh() {

	root := gomu.playlist.GetRoot()

	prevFileName := gomu.playlist.GetCurrentNode().GetText()

	root.ClearChildren()

	populate(root, root.GetReference().(*AudioFile).path)

	root.Walk(func(node, _ *tview.TreeNode) bool {

		// to preserve previously highlighted node
		if node.GetText() == prevFileName {
			p.setHighlight(node)
			return false
		}

		return true
	})

}

// Adds child while setting reference to audio file
func (p *Playlist) addSongToPlaylist(
	audioPath string, selPlaylist *tview.TreeNode,
) error {

	f, err := os.Open(audioPath)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer f.Close()

	songName := getName(audioPath)
	node := tview.NewTreeNode(songName)

	audioLength, err := getLength(audioPath)
	if err != nil {
		return tracerr.Wrap(err)
	}

	audioFile := &AudioFile{
		name:        songName,
		path:        audioPath,
		isAudioFile: true,
		length:      audioLength,
		node:        node,
		parent:      selPlaylist,
	}

	displayText := setDisplayText(songName)

	node.SetReference(audioFile)
	node.SetText(displayText)
	selPlaylist.AddChild(node)

	return nil
}

// Gets all audio files walks from music root directory
func (p *Playlist) getAudioFiles() []*AudioFile {

	root := p.GetRoot()

	audioFiles := []*AudioFile{}

	root.Walk(func(node, _ *tview.TreeNode) bool {

		audioFile := node.GetReference().(*AudioFile)
		audioFiles = append(audioFiles, audioFile)

		return true
	})

	return audioFiles
}

// Creates a directory under selected node, returns error if playlist exists
func (p *Playlist) createPlaylist(name string) error {

	selectedNode := p.GetCurrentNode()

	parentNode := selectedNode.GetReference().(*AudioFile).parent

	// if the current node is the root
	// sets the parent to itself
	if parentNode == nil {
		parentNode = selectedNode
	}

	audioFile := parentNode.GetReference().(*AudioFile)

	err := os.Mkdir(path.Join(audioFile.path, name), 0744)

	if err != nil {
		return tracerr.Wrap(err)
	}

	p.refresh()

	return nil

}

// This is used to replace default behaviour of SetCurrentNode which
// adds color highlight attributes
func (p *Playlist) setHighlight(currNode *tview.TreeNode) {

	if p.prevNode != nil {
		p.prevNode.SetColor(gomu.colors.background)
	}
	currNode.SetColor(gomu.colors.accent)
	p.SetCurrentNode(currNode)

	if currNode.GetReference().(*AudioFile).isAudioFile {
		p.prevNode = currNode
	}

}

// Traverses the playlist and finds the AudioFile struct
// audioName must be hashed with sha1 first
func (p *Playlist) findAudioFile(audioName string) (*AudioFile, error) {

	root := p.GetRoot()

	var selNode *AudioFile

	root.Walk(func(node, _ *tview.TreeNode) bool {

		audioFile := node.GetReference().(*AudioFile)

		hashed := sha1Hex(getName(audioFile.name))

		if hashed == audioName {
			selNode = audioFile
			return false
		}

		return true
	})

	if selNode == nil {
		return nil, tracerr.New("no matching audio name")
	}

	return selNode, nil
}

func (p *Playlist) rename(newName string) error {

	currentNode := p.GetCurrentNode()
	audio := currentNode.GetReference().(*AudioFile)
	pathToFile, _ := filepath.Split(audio.path)
	var newPath string
	if audio.isAudioFile {
		newPath = pathToFile + newName + ".mp3"
	} else {
		newPath = pathToFile + newName
	}
	err := os.Rename(audio.path, newPath)
	if err != nil {
		return tracerr.Wrap(err)
	}

	audio.path = newPath
	gomu.queue.saveQueue(false)
	gomu.queue.clearQueue()
	gomu.queue.loadQueue()

	return nil
}

// updateTitle creates a spinning motion on the title
// of the playlist panel when downloading.
func (p *Playlist) updateTitle() {

	if p.download == 0 {
		p.SetTitle(p.defaultTitle)
		return
	}

	// only one call can be made in one time
	if p.download > 1 {
		return
	}

	s := spin.New()

Download:
	for {

		select {
		case <-p.done:
			p.download -= 1
			if p.download == 0 {
				p.SetTitle(p.defaultTitle)
				break Download
			}
		case <-time.After(time.Millisecond * 100):

			r, g, b := gomu.colors.accent.RGB()
			hexColor := padHex(r, g, b)

			title := fmt.Sprintf("─ Playlist ──┤ %d downloads [green]%s[#%s] ├",
				p.download, s.Next(), hexColor)
			p.SetTitle(title)
			gomu.app.Draw()
		}
	}

}

// Download audio from youtube audio and adds the song to the selected playlist
func ytdl(url string, selPlaylist *tview.TreeNode) error {

	// lookup if youtube-dl exists
	_, err := exec.LookPath("youtube-dl")

	if err != nil {
		defaultTimedPopup(" Error ", "youtube-dl is not in your $PATH")

		return tracerr.Wrap(err)
	}

	selAudioFile := selPlaylist.GetReference().(*AudioFile)
	dir := selAudioFile.path

	defaultTimedPopup(" Ytdl ", "Downloading")

	// specify the output path for ytdl
	outputDir := fmt.Sprintf(
		"%s/%%(title)s.%%(ext)s",
		dir)

	metaData := fmt.Sprintf("%%(artist)s - %%(title)s")

	args := []string{
		"--extract-audio",
		"--audio-format",
		"mp3",
		"--output",
		outputDir,
		"--add-metadata",
		"--embed-thumbnail",
		"--metadata-from-title",
		metaData,
		"--write-sub",
		"--all-subs",
		"--convert-subs",
		"srt",
		// "--cookies",
		// "~/Downloads/youtube.com_cookies.txt",
		url,
	}

	cmd := exec.Command("youtube-dl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	gomu.playlist.download++
	go gomu.playlist.updateTitle()

	// blocking
	err = cmd.Run()

	gomu.playlist.done <- struct{}{}

	if err != nil {
		errorPopup(err)
		return tracerr.Wrap(err)
	}

	playlistPath := dir
	audioPath := extractFilePath(stdout.Bytes(), playlistPath)

	historyPath := gomu.anko.GetString("General.history_path")

	err = appendFile(expandTilde(historyPath), url+"\n")
	if err != nil {
		return tracerr.Wrap(err)
	}

	err = gomu.playlist.addSongToPlaylist(audioPath, selPlaylist)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// Embed lyric to mp3 as uslt
	var tag *id3v2.Tag
	tag, err = id3v2.Open(audioPath, id3v2.Options{Parse: true})
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer tag.Close()

	// langLyric := gomu.anko.GetString("General.lang_lyric")

	pathToFile, _ := filepath.Split(audioPath)
	files, err := ioutil.ReadDir(pathToFile)
	if err != nil {
		logError(err)
	}
	var lyricWritten int = 0
	for _, file := range files {
		// songNameWithoutExt := getName(audioPath)
		fileName := file.Name()
		fileExt := filepath.Ext(fileName)
		lyricFileName := filepath.Join(pathToFile, fileName)
		if fileExt == ".srt" {
			// Embed all lyrics and use langExt as content descriptor of uslt
			fileNameWithoutExt := strings.TrimSuffix(fileName, fileExt)
			langExt := strings.TrimPrefix(filepath.Ext(fileNameWithoutExt), ".")

			// Read entire file content, giving us little control but
			// making it very simple. No need to close the file.
			byteContent, err := ioutil.ReadFile(lyricFileName)
			if err != nil {
				return tracerr.Wrap(err)
			}
			lyricContent := string(byteContent)

			err = embedLyric(audioPath, lyricContent, langExt)
			if err != nil {
				return tracerr.Wrap(err)
			}
			err = os.Remove(lyricFileName)
			if err != nil {
				return tracerr.Wrap(err)
			}
			lyricWritten++
		}
	}

	downloadFinishedMessage := fmt.Sprintf("Finished downloading\n%s\n%v lyrics embeded", getName(audioPath), lyricWritten)
	defaultTimedPopup(" Ytdl ", downloadFinishedMessage)
	gomu.app.Draw()

	return nil
}

// Add songs and their directories in Playlist panel
func populate(root *tview.TreeNode, rootPath string) error {

	files, err := ioutil.ReadDir(rootPath)

	if err != nil {
		return tracerr.Wrap(err)
	}

	for _, file := range files {

		path, err := filepath.EvalSymlinks(filepath.Join(rootPath, file.Name()))
		if err != nil {
			continue
		}

		songName := getName(file.Name())
		child := tview.NewTreeNode(songName)

		if file.Mode().IsRegular() {

			f, err := os.Open(path)
			if err != nil {
				continue
			}
			defer f.Close()

			filetype, err := getFileContentType(f)

			if err != nil {
				continue
			}

			// skip if not mp3 file
			if filetype != "mpeg" {
				continue
			}

			audioFile := &AudioFile{
				name:        songName,
				path:        path,
				isAudioFile: true,
				node:        child,
				parent:      root,
			}

			displayText := setDisplayText(songName)

			child.SetReference(audioFile)
			child.SetText(displayText)
			root.AddChild(child)

		}

		if file.IsDir() || file.Mode()&os.ModeSymlink != 0 {

			audioFile := &AudioFile{
				name:        songName,
				path:        path,
				isAudioFile: false,
				node:        child,
				parent:      root,
			}

			displayText := setDisplayText(songName)

			child.SetReference(audioFile)
			child.SetColor(gomu.colors.accent)
			child.SetText(displayText)
			root.AddChild(child)
			populate(child, path)

		}

	}

	return nil
}

func (p *Playlist) yank() error {
	yankFile = p.getCurrentFile()
	if yankFile == nil {
		isYanked = false
		defaultTimedPopup(" Error! ", "No file has been yanked.")
		return nil
	}
	if yankFile.node == p.GetRoot() {
		isYanked = false
		defaultTimedPopup(" Error! ", "Please don't yank the root directory.")
		return nil
	}
	isYanked = true
	defaultTimedPopup(" Success ", yankFile.name+"\n has been yanked successfully.")

	return nil
}

func (p *Playlist) paste() error {
	if isYanked {
		isYanked = false
		oldPathDir, oldPathFileName := filepath.Split(yankFile.path)
		pasteFile := p.getCurrentFile()
		if pasteFile.isAudioFile {
			newPathDir, _ := filepath.Split(pasteFile.path)
			if oldPathDir == newPathDir {
				return nil
			}
			newPathFull := filepath.Join(newPathDir, oldPathFileName)
			err := os.Rename(yankFile.path, newPathFull)
			if err != nil {
				defaultTimedPopup(" Error ", yankFile.name+"\n has not been pasted.")
				return tracerr.Wrap(err)
			}
			defaultTimedPopup(" Success ", yankFile.name+"\n has been pasted to\n"+pasteFile.name)

		} else {
			newPathDir := pasteFile.path
			if oldPathDir == newPathDir {
				return nil
			}
			newPathFull := filepath.Join(newPathDir, oldPathFileName)
			err := os.Rename(yankFile.path, newPathFull)
			if err != nil {
				defaultTimedPopup(" Error ", yankFile.name+"\n has not been pasted.")
				return tracerr.Wrap(err)
			}
			defaultTimedPopup(" Success ", yankFile.name+"\n has been pasted to\n"+pasteFile.name)

		}

		p.refresh()
		gomu.queue.updateQueueNames()
	}

	return nil
}

func setDisplayText(songName string) string {
	useEmoji := gomu.anko.GetBool("General.use_emoji")
	if !useEmoji {
		return songName
	}

	emojiFile := gomu.anko.GetString("Emoji.file")
	return fmt.Sprintf(" %s %s", emojiFile, songName)
}

// populateAudioLength is the most time consuming part of startup,
// so here we initialize it separately
func populateAudioLength(root *tview.TreeNode) error {
	root.Walk(func(node *tview.TreeNode, _ *tview.TreeNode) bool {
		audioFile := node.GetReference().(*AudioFile)
		if audioFile.isAudioFile {
			audioLength, err := getLength(audioFile.path)
			if err != nil {
				logError(err)
				return false
			}
			audioFile.length = audioLength
		}
		return true
	})

	gomu.queue.updateTitle()
	return nil
}
