package main

import (
	"testing"

	"github.com/issadarkthing/gomu/player"
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

	q := newQueue()

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

func TestQueuePrev(t *testing.T) {

	q := newQueue()

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

	q := newQueue()

	for _, v := range sample {
		q.AddItem(v, "", 0, nil)
	}

	initLen := q.GetItemCount()
	q.deleteItem(-1)
	finalLen := q.GetItemCount()

	if initLen != finalLen {
		t.Errorf("Item removed when -1 index was given")
	}

}

func TestUpdateTitle(t *testing.T) {

	gomu := prepareTest()
	audioFiles := gomu.playlist.getAudioFiles()

	for _, v := range audioFiles {
		gomu.queue.enqueue(v)
	}

	expected := gomu.queue.updateTitle()
	got := gomu.queue.GetTitle()

	if expected != got {
		t.Errorf("Expected %s; got %s", expected, got)
	}
}

func TestPushFront(t *testing.T) {

	gomu = prepareTest()
	rapPlaylist := gomu.playlist.GetRoot().GetChildren()[1]

	gomu.playlist.addAllToQueue(rapPlaylist)

	selSong, err := gomu.queue.deleteItem(2)
	if err != nil {
		t.Error(err)
	}

	gomu.queue.pushFront(selSong)

	for i, v := range gomu.queue.items {

		if v == selSong && i != 0 {
			t.Errorf("Item does not move to the 0th index")
		}

	}

}

func TestDequeue(t *testing.T) {

	gomu := prepareTest()

	audioFiles := gomu.playlist.getAudioFiles()

	for _, v := range audioFiles {
		gomu.queue.enqueue(v)
	}

	initLen := len(gomu.queue.items)

	gomu.queue.dequeue()

	finalLen := len(gomu.queue.items)

	if initLen-1 != finalLen {
		t.Errorf("Expected %d got %d", initLen-1, finalLen)
	}

}

func TestEnqueue(t *testing.T) {

	gomu = prepareTest()

	var audioFiles []*player.AudioFile

	gomu.playlist.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {

		audioFile := node.GetReference().(*player.AudioFile)

		if audioFile.IsAudioFile() {
			audioFiles = append(audioFiles, audioFile)
			return false
		}

		return true
	})

	for _, v := range audioFiles {
		gomu.queue.enqueue(v)
	}

	queue := gomu.queue.getItems()

	for i, audioFile := range audioFiles {

		if queue[i] != audioFile.Path() {
			t.Errorf("Invalid path; expected %s got %s", audioFile.Path(), queue[i])
		}
	}

	queueLen := gomu.queue.GetItemCount()

	if queueLen != len(audioFiles) {
		t.Errorf("Invalid count in queue; expected %d, got %d", len(audioFiles), queueLen)
	}

}

func TestQueueGetItems(t *testing.T) {

	q := newQueue()

	for k, v := range sample {
		q.AddItem(k, v, 0, nil)
	}

	got := q.getItems()

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

func TestClearQueue(t *testing.T) {

	gomu = prepareTest()
	rapPlaylist := gomu.playlist.GetRoot().GetChildren()[1]
	gomu.playlist.addAllToQueue(rapPlaylist)

	gomu.queue.clearQueue()

	queueLen := len(gomu.queue.items)
	if queueLen != 0 {
		t.Errorf("Expected %d; got %d", 0, queueLen)
	}

	listLen := len(gomu.queue.getItems())
	if listLen != 0 {
		t.Errorf("Expected %d; got %d", 0, listLen)
	}

}

func TestShuffle(t *testing.T) {

	gomu = prepareTest()

	root := gomu.playlist.GetRoot()
	rapDir := root.GetChildren()[1]

	gomu.playlist.addAllToQueue(rapDir)

	sameCounter := 0
	const limit int = 10

	for i := 0; i < limit; i++ {
		items := gomu.queue.getItems()

		gomu.queue.shuffle()

		got := gomu.queue.getItems()

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
