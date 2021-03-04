package hook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddHook(t *testing.T) {

	h := NewEventHook()

	h.AddHook("a", nil)
	h.AddHook("a", nil)
	h.AddHook("a", nil)
	h.AddHook("a", nil)

	assert.Equal(t, 1, len(h.events), "should only contain 1 event")

	hooks := h.events["a"]
	assert.Equal(t, 4, len(hooks), "should contain 4 hooks")

	h.AddHook("b", nil)
	h.AddHook("c", nil)

	assert.Equal(t, 3, len(h.events), "should contain 3 events")
}

func TestRunHooks(t *testing.T) {

	h := NewEventHook()
	x := 0

	for i := 0; i < 100; i++ {
		h.AddHook("sample", func() {
			x++
		})
	}

	h.AddHook("noop", func() {
		x++
	})

	h.AddHook("noop", func() {
		x++
	})

	assert.Equal(t, x, 0, "should not execute any hook")

	h.RunHooks("x")

	assert.Equal(t, x, 0, "should not execute any hook")

	h.RunHooks("sample")

	assert.Equal(t, x, 100, "should only execute event 'sample'")
}
