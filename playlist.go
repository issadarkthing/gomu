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

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

// Playlist and mp3 files are represented with this struct
// if isAudioFile equals to false it is a directory
type AudioFile struct {
	name        string
	path        string
	isAudioFile bool
	length      time.Duration
	node        *tview.TreeNode
	parent      *tview.TreeNode
}

// Treeview of a music directory
type Playlist struct {
	*tview.TreeView
	prevNode *tview.TreeNode
}

func (p *Playlist) help() []string {

	return []string{
		"j      down",
		"k      up",
		"h      close node",
		"l      add song to queue",
		"L      add playlist to queue",
		"d      delete file from filesystem",
		"D      delete playlist from filesystem",
		"Y      download audio",
		"r      refresh",
		"f      find in playlist",
	}

}

func newPlaylist() *Playlist {

	rootDir, err := filepath.Abs(expandTilde(viper.GetString("music_dir")))

	if err != nil {
		logError(err)
	}

	root := tview.NewTreeNode(path.Base(rootDir)).
		SetColor(gomu.accentColor)

	tree := tview.NewTreeView().SetRoot(root)

	playlist := &Playlist{
		TreeView: tree,
	}

	rootAudioFile := &AudioFile{
		name: root.GetText(),
		node: root,
		path: rootDir,
	}

	root.SetReference(rootAudioFile)

	playlist.
		SetTitle(" Playlist ").
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft)

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

		currNode := playlist.GetCurrentNode()
		audioFile := currNode.GetReference().(*AudioFile)

		switch e.Rune() {

		case ' ':
			// Disable default key handler
			return nil

		case 'a':

			name, _ := gomu.pages.GetFrontPage()
			if name != "mkdir-popup" {
				createPlaylistPopup()
			}

		case 'D':

			err := playlist.deletePlaylist(audioFile)
			if err != nil {
				logError(err)
			}

		case 'd':

			// prevent from deleting a directory
			if !audioFile.isAudioFile {
				return e
			}

			err := playlist.deleteSong(audioFile)
			if err != nil {
				logError(err)
			}

		case 'Y':

			if gomu.pages.HasPage("download-popup") {
				gomu.pages.RemovePage("download-popup")
				return e
			}

			// this ensures it downloads to
			// the correct dir
			if audioFile.isAudioFile {
				downloadMusicPopup(audioFile.parent)
			} else {
				downloadMusicPopup(currNode)
			}

		case 'l':

			if audioFile.isAudioFile {
				gomu.queue.enqueue(audioFile)
			} else {
				currNode.SetExpanded(true)
			}

		case 'h':

			// if closing node with no children
			// close the node's parent
			// remove the color of the node

			if audioFile.isAudioFile {
				parent := audioFile.parent

				playlist.setHighlight(parent)

				parent.SetExpanded(false)
			}

			currNode.Collapse()

		case 'L':

			if !viper.GetBool("confirm_bulk_add") {
				playlist.addAllToQueue(playlist.GetCurrentNode())
				return e
			}

			confirmationPopup(
				"Are you sure to add this whole directory into queue?",
				func(_ int, label string) {

					if label == "yes" {
						playlist.addAllToQueue(playlist.GetCurrentNode())
					}

					gomu.pages.RemovePage("confirmation-popup")
					gomu.app.SetFocus(playlist)

				})

		case 'r':

			playlist.refresh()

		case 'f':

			err := playlist.fuzzyFind()
			if err != nil {
				logError(err)
			}

		}

		return e
	})

	return playlist

}

// Deletes song from filesystem
func (p *Playlist) deleteSong(audioFile *AudioFile) (err error) {

	confirmationPopup(
		"Are you sure to delete this audio file?", func(_ int, buttonName string) {

			if buttonName == "no" || buttonName == "" {
				gomu.pages.RemovePage("confirmation-popup")
				gomu.popups.pop()
				return
			}

			err := os.Remove(audioFile.path)

			if err != nil {

				timedPopup(" Error ", "Unable to delete "+audioFile.name,
					getPopupTimeout(), 0, 0)

				err = tracerr.Wrap(err)

			} else {

				timedPopup(" Success ", audioFile.name+"\nhas been deleted successfully",
					getPopupTimeout(), 0, 0)

				p.refresh()
			}

			gomu.pages.RemovePage("confirmation-popup")
			gomu.popups.pop()
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
				gomu.pages.RemovePage("confirmation-popup")
				gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))
				return
			}

			err := os.RemoveAll(selectedDir.path)

			if err != nil {

				timedPopup(
					" Error ",
					"Unable to delete dir "+selectedDir.name,
					getPopupTimeout(), 0, 0)

				err = tracerr.Wrap(err)

			} else {

				timedPopup(
					" Success ",
					selectedDir.name+"\nhas been deleted successfully",
					getPopupTimeout(), 0, 0)

				p.refresh()
			}

			gomu.pages.RemovePage("confirmation-popup")
			gomu.app.SetFocus(gomu.prevPanel.(tview.Primitive))

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
		gomu.queue.enqueue(currNode)
	}

}

// Refreshes the playlist and read the whole root music dir
func (p *Playlist) refresh() {

	root := gomu.playlist.GetRoot()

	prevFileName := gomu.playlist.GetCurrentNode().GetText()

	root.ClearChildren()

	populate(root, root.GetReference().(*AudioFile).path)

	root.Walk(func(node, parent *tview.TreeNode) bool {

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

	node := tview.NewTreeNode(getName(audioPath))

	audioLength, err := getLength(audioPath)

	if err != nil {
		return tracerr.Wrap(err)
	}

	audioFile := &AudioFile{
		name:        path.Base(audioPath),
		path:        audioPath,
		isAudioFile: true,
		length:      audioLength,
		parent:      selPlaylist,
	}

	node.SetReference(audioFile)
	selPlaylist.AddChild(node)
	gomu.app.Draw()

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
		p.prevNode.SetColor(gomu.textColor)
	}
	currNode.SetColor(gomu.accentColor)
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

// Highlight the selected node searched using fzf
func (p *Playlist) fuzzyFind() error {

	var result string
	var err error

	audioFiles := p.getAudioFiles()
	paths := make(map[string]*tview.TreeNode, len(audioFiles))
	input := make([]string, 0, len(audioFiles))

	for _, v := range audioFiles {
		rootDir := audioFiles[0].path + "/"
		// path relative to music directory
		shortPath := strings.TrimPrefix(v.path, rootDir)
		paths[shortPath] = v.node
		input = append(input, shortPath)
	}

	gomu.suspend()
	ok := gomu.app.Suspend(func() {
		res, e := fzfFind(input)
		if e != nil {
			err = tracerr.Wrap(e)
		}
		result = res
	})
	gomu.unsuspend()

	if err != nil {
		return tracerr.Wrap(err)
	}

	if !ok {
		return tracerr.New("App was not suspended")
	}

	if result == "" {
		return nil
	}

	if err != nil {
		return tracerr.Wrap(err)
	}

	var selNode *tview.TreeNode
	selNode = paths[result]
	p.setHighlight(selNode)

	return nil

}

// Takes a list of input and suspends tview
// returns empty string if cancelled
func fzfFind(input []string) (string, error) {

	var in strings.Builder
	var out strings.Builder

	for _, v := range input {
		in.WriteString(v + "\n")
	}

	cmd := exec.Command("fzf")
	cmd.Stdin = strings.NewReader(in.String())
	cmd.Stderr = os.Stderr
	cmd.Stdout = &out

	if err := cmd.Run(); cmd.ProcessState.ExitCode() == 130 {
		// exit code 130 is when we cancel FZF
		// not an error
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("failed to find a file: %s", err)
	}

	f := strings.TrimSpace(out.String())

	return f, nil
}

// Download audio from youtube audio and adds the song to the selected playlist
func ytdl(url string, selPlaylist *tview.TreeNode) error {

	// lookup if youtube-dl exists
	_, err := exec.LookPath("youtube-dl")

	if err != nil {
		timedPopup(" Error ", "youtube-dl is not in your $PATH",
			getPopupTimeout(), 0, 0)

		return tracerr.Wrap(err)
	}

	dir := viper.GetString("music_dir")

	selAudioFile := selPlaylist.GetReference().(*AudioFile)
	selPlaylistName := selAudioFile.name

	timedPopup(" Ytdl ", "Downloading", getPopupTimeout(), 0, 0)

	// specify the output path for ytdl
	outputDir := fmt.Sprintf(
		"%s/%s/%%(title)s.%%(ext)s",
		dir,
		selPlaylistName)

	args := []string{
		"--extract-audio",
		"--audio-format",
		"mp3",
		"--output",
		outputDir,
		url,
	}

	cmd := exec.Command("youtube-dl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if err != nil {
		timedPopup(" Error ", "Error running youtube-dl", getPopupTimeout(), 0, 0)
		return tracerr.Wrap(err)
	}

	playlistPath := path.Join(expandTilde(dir), selPlaylistName)
	audioPath := extractFilePath(stdout.Bytes(), playlistPath)

	err = gomu.playlist.addSongToPlaylist(audioPath, selPlaylist)

	if err != nil {
		return tracerr.Wrap(err)
	}

	downloadFinishedMessage := fmt.Sprintf("Finished downloading\n%s",
		path.Base(audioPath))

	timedPopup(
		" Ytdl ",
		downloadFinishedMessage,
		getPopupTimeout(), 0, 0)

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

		path := filepath.Join(rootPath, file.Name())
		f, err := os.Open(path)

		if err != nil {
			continue
		}

		defer f.Close()

		songName := getName(file.Name())
		child := tview.NewTreeNode(songName)

		if !file.IsDir() {

			filetype, err := getFileContentType(f)

			if err != nil {
				continue
			}

			// skip if not mp3 file
			if filetype != "mpeg" {
				continue
			}

			audioLength, err := getLength(path)

			if err != nil {
				continue
			}

			audioFile := &AudioFile{
				name:        songName,
				path:        path,
				isAudioFile: true,
				length:      audioLength,
				node:        child,
				parent:      root,
			}

			child.SetReference(audioFile)
			root.AddChild(child)

		}

		if file.IsDir() {

			audioFile := &AudioFile{
				name:   songName,
				path:   path,
				node:   child,
				parent: root,
			}
			child.SetReference(audioFile)
			child.SetColor(gomu.accentColor)
			root.AddChild(child)
			populate(child, path)

		}

	}

	return nil
}
