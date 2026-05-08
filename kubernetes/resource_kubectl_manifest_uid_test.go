package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveUIDForPlannedState(t *testing.T) {
	t.Run("returns updated uid when created and live uid differ", func(t *testing.T) {
		uid, shouldUpdate := resolveUIDForPlannedState("uid-created", "uid-live", true)
		assert.True(t, shouldUpdate)
		assert.Equal(t, "uid-live", uid)
	})

	t.Run("does not update when uids are equal", func(t *testing.T) {
		uid, shouldUpdate := resolveUIDForPlannedState("uid-same", "uid-same", true)
		assert.False(t, shouldUpdate)
		assert.Equal(t, "", uid)
	})

	t.Run("does not update when created uid missing", func(t *testing.T) {
		uid, shouldUpdate := resolveUIDForPlannedState("", "uid-live", true)
		assert.False(t, shouldUpdate)
		assert.Equal(t, "", uid)
	})

	t.Run("does not update when live uid missing", func(t *testing.T) {
		uid, shouldUpdate := resolveUIDForPlannedState("uid-created", "", true)
		assert.False(t, shouldUpdate)
		assert.Equal(t, "", uid)
	})

	t.Run("does not update when live uid is unknown in diff", func(t *testing.T) {
		uid, shouldUpdate := resolveUIDForPlannedState("uid-created", "uid-live", false)
		assert.False(t, shouldUpdate)
		assert.Equal(t, "", uid)
	})

}
