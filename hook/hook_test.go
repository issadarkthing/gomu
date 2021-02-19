package hook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunHook(t *testing.T) {

	h := NewEventHook()
	i := 0

	h.AddHook("sample", func() {
		i++
	})

	h.AddHook("sample", func() {
		i++
	})

	assert.Equal(t, i, 0)

	h.RunHooks("x")

	assert.Equal(t, i, 0)

	h.RunHooks("sample")

	assert.Equal(t, i, 2)
}
