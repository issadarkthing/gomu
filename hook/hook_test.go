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
	i := 0

	h.AddHook("sample", func() {
		i++
	})

	h.AddHook("sample", func() {
		i++
	})

	h.AddHook("noop", func() {
		i++
	})

	assert.Equal(t, i, 0, "should not execute any hook")

	h.RunHooks("x")

	assert.Equal(t, i, 0, "should not execute any hook")

	h.RunHooks("sample")

	assert.Equal(t, i, 2, "should only execute event 'sample'")
}
