package main

import (
	"testing"

	"github.com/rivo/tview"
)

var sample = map[string]string{
	"a": "1",
	"b": "2",
	"c": "3",
	"d": "4",
	"e": "5",
}

func TestQueueNext(t *testing.T) {

	q := NewQueue()

	for _, v := range sample {
		q.AddItem(v, "", 0, nil)
	}

	q.SetCurrentItem(0)
	q.next()

	got := q.GetCurrentItem()

	if got != 1 {
		t.Errorf("Expected %d got %d", 1, got)
	}

}

func TestDequeue(t *testing.T) {

	gomu := prepareTest()

	audioFiles := gomu.Playlist.GetAudioFiles()

	for _, v := range audioFiles {
		gomu.Queue.Enqueue(v)
	}

	initLen := len(gomu.Queue.Items)

	gomu.Queue.Dequeue()

	finalLen := len(gomu.Queue.Items)

	if initLen-1 != finalLen {
		t.Errorf("Expected %d got %d", initLen-1, finalLen)
	}

}

func TestQueuePrev(t *testing.T) {

	q := NewQueue()

	for _, v := range sample {
		q.AddItem(v, "", 0, nil)
	}

	q.SetCurrentItem(3)
	q.prev()

	got := q.GetCurrentItem()

	if got != 2 {
		t.Errorf("Expected %d got %d", 1, got)
	}

}

func TestQueueDeleteItem(t *testing.T) {

	q := NewQueue()

	for _, v := range sample {
		q.AddItem(v, "", 0, nil)
	}

	initLen := q.GetItemCount()
	q.DeleteItem(-1)
	finalLen := q.GetItemCount()

	if initLen != finalLen {
		t.Errorf("Item removed when -1 index was given")
	}

}

func TestEnqueue(t *testing.T) {

	gomu = prepareTest()

	var audioFiles []*AudioFile

	gomu.Playlist.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {

		audioFile := node.GetReference().(*AudioFile)

		if audioFile.IsAudioFile {
			audioFiles = append(audioFiles, audioFile)
			return false
		}

		return true
	})

	for _, v := range audioFiles {
		gomu.Queue.Enqueue(v)
	}

	queue := gomu.Queue.GetItems()

	for i, audioFile := range audioFiles {

		if queue[i] != audioFile.Path {
			t.Errorf("Invalid path; expected %s got %s", audioFile.Path, queue[i])
		}
	}

	queueLen := gomu.Queue.GetItemCount()

	if queueLen != len(audioFiles) {
		t.Errorf("Invalid count in queue; expected %d, got %d", len(audioFiles), queueLen)
	}

}

func TestQueueGetItems(t *testing.T) {

	q := NewQueue()

	for k, v := range sample {
		q.AddItem(k, v, 0, nil)
	}

	got := q.GetItems()

	if len(got) != len(sample) {
		t.Errorf("GetItems does not return correct items length")
	}

	sampleValues := []string{}

	for _, v := range sample {
		sampleValues = append(sampleValues, v)
	}

	for _, v := range got {
		if !SliceHas(v, sampleValues) {
			t.Error("GetItems does not return correct items")
		}
	}

}

func TestPushFront(t *testing.T) {

	gomu = prepareTest()
	rapPlaylist := gomu.Playlist.GetRoot().GetChildren()[1]
	gomu.Playlist.AddAllToQueue(rapPlaylist)

	selSong := gomu.Queue.DeleteItem(2)

	gomu.Queue.PushFront(selSong)

	for i, v := range gomu.Queue.Items {

		if v == selSong && i != 0 {
			t.Errorf("Item does not move to the 0th index")
		}

	}

}

func TestClearQueue(t *testing.T) {

	gomu = prepareTest()
	rapPlaylist := gomu.Playlist.GetRoot().GetChildren()[1]
	gomu.Playlist.AddAllToQueue(rapPlaylist)

	gomu.Queue.ClearQueue()

	queueLen := len(gomu.Queue.Items)
	if queueLen != 0 {
		t.Errorf("Expected %d; got %d", 0, queueLen)
	}

	listLen := len(gomu.Queue.GetItems())
	if listLen != 0 {
		t.Errorf("Expected %d; got %d", 0, listLen)
	}

}

func TestShuffle(t *testing.T) {

	gomu = prepareTest()

	root := gomu.Playlist.GetRoot()
	rapDir := root.GetChildren()[1]

	gomu.Playlist.AddAllToQueue(rapDir)

	sameCounter := 0
	const limit int = 10

	for i := 0; i < limit; i++ {
		items := gomu.Queue.GetItems()

		gomu.Queue.Shuffle()

		got := gomu.Queue.GetItems()

		if Equal(items, got) {
			sameCounter++
		}
	}

	if sameCounter == limit {
		t.Error("Items in queue are not changed")
	}

}

// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func Equal(a, b []string) bool {
    if len(a) != len(b) {
        return false
    }
    for i, v := range a {
        if v != b[i] {
            return false
        }
    }
    return true
}

// utility function to check elem in a slice
func SliceHas(item string, s []string) bool {

	for _, v := range s {
		if v == item {
			return true
		}
	}

	return false
}
