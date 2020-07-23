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

type AudioFile struct {
	Name        string
	Path        string
	IsAudioFile bool
	Length      time.Duration
	Node        *tview.TreeNode
	Parent      *tview.TreeNode
}

type Playlist struct {
	*tview.TreeView
	prevNode *tview.TreeNode
}

func (p *Playlist) Help() []string {

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

func NewPlaylist() *Playlist {

	rootDir, err := filepath.Abs(expandTilde(viper.GetString("music_dir")))

	if err != nil {
		LogError(err)
	}

	root := tview.NewTreeNode(path.Base(rootDir)).
		SetColor(gomu.AccentColor)

	tree := tview.NewTreeView().SetRoot(root)

	playlist := &Playlist{
		TreeView: tree,
	}

	rootAudioFile := &AudioFile{
		Name: root.GetText(),
		Node: root,
		Path: rootDir,
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

	playlist.SetHighlight(firstChild)

	playlist.SetChangedFunc(func(node *tview.TreeNode) {
		playlist.SetHighlight(node)
	})

	playlist.SetSelectedFunc(func(node *tview.TreeNode) {
		node.SetExpanded(!node.IsExpanded())
	})

	playlist.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		currNode := playlist.GetCurrentNode()

		audioFile := currNode.GetReference().(*AudioFile)

		switch e.Rune() {

		case 'a':

			name, _ := gomu.Pages.GetFrontPage()

			if name != "mkdir-popup" {
				CreatePlaylistPopup()
			}

		case 'D':

			err := playlist.DeletePlaylist(audioFile)
			if err != nil {
				LogError(err)
			}

		case 'd':

			// prevent from deleting a directory
			if !audioFile.IsAudioFile {
				return e
			}

			err := playlist.DeleteSong(audioFile)
			if err != nil {
				LogError(err)
			}

		case 'Y':

			if gomu.Pages.HasPage("download-popup") {
				gomu.Pages.RemovePage("download-popup")
				return e
			}

			// this ensures it downloads to
			// the correct dir
			if audioFile.IsAudioFile {
				downloadMusicPopup(audioFile.Parent)
			} else {
				downloadMusicPopup(currNode)
			}

		case 'l':

			if audioFile.IsAudioFile {
				gomu.Queue.Enqueue(audioFile)
			} else {
				currNode.SetExpanded(true)
			}

		case 'h':

			// if closing node with no children
			// close the node's parent
			// remove the color of the node

			if audioFile.IsAudioFile {
				parent := audioFile.Parent

				playlist.SetHighlight(parent)

				parent.SetExpanded(false)
			}

			currNode.Collapse()

		case 'L':

			if !viper.GetBool("confirm_bulk_add") {
				playlist.AddAllToQueue(playlist.GetCurrentNode())
				return e
			}

			confirmationPopup(
				"Are you sure to add this whole directory into queue?",
				func(_ int, label string) {

					if label == "yes" {
						playlist.AddAllToQueue(playlist.GetCurrentNode())
					}

					gomu.Pages.RemovePage("confirmation-popup")
					gomu.App.SetFocus(playlist)

				})

		case 'r':

			playlist.Refresh()

		case 'f':

			err := playlist.FuzzyFind()
			if err != nil {
				LogError(err)
			}

		}

		return e
	})

	return playlist

}

func (p *Playlist) DeleteSong(audioFile *AudioFile) (err error) {

	confirmationPopup(
		"Are you sure to delete this audio file?", func(_ int, buttonName string) {

			if buttonName == "no" || buttonName == "" {
				gomu.Pages.RemovePage("confirmation-popup")
				gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
				return
			}

			err := os.Remove(audioFile.Path)

			if err != nil {

				timedPopup(" Error ", "Unable to delete "+audioFile.Name,
					getPopupTimeout(), 0, 0)

				err = tracerr.Wrap(err)

			} else {

				timedPopup(" Success ", audioFile.Name+"\nhas been deleted successfully",
					getPopupTimeout(), 0, 0)

				p.Refresh()
			}

			gomu.Pages.RemovePage("confirmation-popup")
			gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
		})

	return nil
}

// Deletes playlist/dir from filesystem
func (p *Playlist) DeletePlaylist(audioFile *AudioFile) (err error) {

	var selectedDir *AudioFile

	// gets the parent dir if current focused node is not a dir
	if audioFile.IsAudioFile {
		selectedDir = audioFile.Parent.GetReference().(*AudioFile)
	} else {
		selectedDir = audioFile
	}

	confirmationPopup("Are you sure to delete this directory?",
		func(_ int, buttonName string) {

			if buttonName == "no" || buttonName == "" {
				gomu.Pages.RemovePage("confirmation-popup")
				gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))
				return
			}

			err := os.RemoveAll(selectedDir.Path)

			if err != nil {

				timedPopup(
					" Error ",
					"Unable to delete dir "+selectedDir.Name,
					getPopupTimeout(), 0, 0)

				err = tracerr.Wrap(err)

			} else {

				timedPopup(
					" Success ",
					selectedDir.Name+"\nhas been deleted successfully",
					getPopupTimeout(), 0, 0)

				p.Refresh()
			}

			gomu.Pages.RemovePage("confirmation-popup")
			gomu.App.SetFocus(gomu.PrevPanel.(tview.Primitive))

		})

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

		songName := GetName(file.Name())
		child := tview.NewTreeNode(songName)

		if !file.IsDir() {

			filetype, err := GetFileContentType(f)

			if err != nil {
				continue
			}

			// skip if not mp3 file
			if filetype != "mpeg" {
				continue
			}

			audioLength, err := GetLength(path)

			if err != nil {
				continue
			}

			audioFile := &AudioFile{
				Name:        songName,
				Path:        path,
				IsAudioFile: true,
				Length:      audioLength,
				Node:        child,
				Parent:      root,
			}

			child.SetReference(audioFile)
			root.AddChild(child)

		}

		if file.IsDir() {

			audioFile := &AudioFile{
				Name:        songName,
				Path:        path,
				IsAudioFile: false,
				Length:      0,
				Node:        child,
				Parent:      root,
			}
			child.SetReference(audioFile)
			child.SetColor(gomu.AccentColor)
			root.AddChild(child)
			populate(child, path)

		}

	}

	return nil

}

// Bulk add a playlist to queue
func (p *Playlist) AddAllToQueue(root *tview.TreeNode) {

	var childrens []*tview.TreeNode
	childrens = root.GetChildren()

	// gets the parent if the highlighted item is a file
	if root.GetReference().(*AudioFile).IsAudioFile {
		childrens = root.GetReference().(*AudioFile).Parent.GetChildren()
	}

	for _, v := range childrens {
		currNode := v.GetReference().(*AudioFile)
		gomu.Queue.Enqueue(currNode)
	}

}

// Refresh the playlist and read the whole root music dir
func (p *Playlist) Refresh() {

	root := gomu.Playlist.GetRoot()

	prevFileName := gomu.Playlist.GetCurrentNode().GetText()

	root.ClearChildren()

	populate(root, root.GetReference().(*AudioFile).Path)

	root.Walk(func(node, parent *tview.TreeNode) bool {

		// to preserve previously highlighted node
		if node.GetText() == prevFileName {
			p.SetHighlight(node)
			return false
		}

		return true
	})

}

// Adds child while setting reference to audio file
func (p *Playlist) AddSongToPlaylist(
	audioPath string, selPlaylist *tview.TreeNode,
) error {

	f, err := os.Open(audioPath)

	if err != nil {
		return tracerr.Wrap(err)
	}

	defer f.Close()

	node := tview.NewTreeNode(GetName(audioPath))

	audioLength, err := GetLength(audioPath)

	if err != nil {
		return tracerr.Wrap(err)
	}

	audioFile := &AudioFile{
		Name:        path.Base(audioPath),
		Path:        audioPath,
		IsAudioFile: true,
		Length:      audioLength,
		Parent:      selPlaylist,
	}

	node.SetReference(audioFile)
	selPlaylist.AddChild(node)
	gomu.App.Draw()

	return nil

}

// Gets all audio files walks from music root directory
func (p *Playlist) GetAudioFiles() []*AudioFile {

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
func (p *Playlist) CreatePlaylist(name string) error {

	selectedNode := p.GetCurrentNode()

	parentNode := selectedNode.GetReference().(*AudioFile).Parent

	// if the current node is the root
	// sets the parent to itself
	if parentNode == nil {
		parentNode = selectedNode
	}

	audioFile := parentNode.GetReference().(*AudioFile)

	err := os.Mkdir(path.Join(audioFile.Path, name), 0744)

	if err != nil {
		return tracerr.Wrap(err)
	}

	p.Refresh()

	return nil

}

// This is used to replace default behaviour of SetCurrentNode which
// adds color highlight attributes
func (p *Playlist) SetHighlight(currNode *tview.TreeNode) {

	if p.prevNode != nil {
		p.prevNode.SetColor(gomu.TextColor)
	}
	currNode.SetColor(gomu.AccentColor)
	p.SetCurrentNode(currNode)

	if currNode.GetReference().(*AudioFile).IsAudioFile {
		p.prevNode = currNode
	}

}

// Traverses the playlist and finds the AudioFile struct
// audioName must be hashed with sha1 first
func (p *Playlist) FindAudioFile(audioName string) (*AudioFile, error) {

	root := p.GetRoot()

	var selNode *AudioFile

	root.Walk(func(node, _ *tview.TreeNode) bool {

		audioFile := node.GetReference().(*AudioFile)

		hashed := Sha1Hex(GetName(audioFile.Name))

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
func (p *Playlist) FuzzyFind() error {

	var result string
	var err error

	audioFiles := p.GetAudioFiles()
	paths := make(map[string]*tview.TreeNode, len(audioFiles))
	input := make([]string, 0, len(audioFiles))

	for _, v := range audioFiles {
		rootDir := audioFiles[0].Path + "/"
		// path relative to music directory
		shortPath := strings.TrimPrefix(v.Path, rootDir)
		paths[shortPath] = v.Node
		input = append(input, shortPath)
	}

	gomu.Suspend()
	ok := gomu.App.Suspend(func() {
		res, e := FzfFind(input)
		if e != nil {
			err = tracerr.Wrap(e)
		}
		result = res
	})
	gomu.Unsuspend()

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
	p.SetHighlight(selNode)

	return nil

}

// Takes a list of input and suspends tview
// returns empty string if cancelled
func FzfFind(input []string) (string, error) {

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

// download audio from youtube audio and adds the song to the selected playlist
func Ytdl(url string, selPlaylist *tview.TreeNode) error {

	// lookup if youtube-dl exists
	_, err := exec.LookPath("youtube-dl")

	if err != nil {
		timedPopup(" Error ", "youtube-dl is not in your $PATH",
			getPopupTimeout(), 0, 0)

		return tracerr.Wrap(err)
	}

	dir := viper.GetString("music_dir")

	selAudioFile := selPlaylist.GetReference().(*AudioFile)
	selPlaylistName := selAudioFile.Name

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

	err = gomu.Playlist.AddSongToPlaylist(audioPath, selPlaylist)

	if err != nil {
		return tracerr.Wrap(err)
	}

	downloadFinishedMessage := fmt.Sprintf("Finished downloading\n%s",
		path.Base(audioPath))

	timedPopup(
		" Ytdl ",
		downloadFinishedMessage,
		getPopupTimeout(), 0, 0)

	gomu.App.Draw()

	return nil

}
