package k3sctl

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/teleproxy/pkg/dlock"
)

func TestContainer(t *testing.T) {
	// we get the lock to make sure we are the only thing running
	// because the nat tests interfere with docker functionality
	assert.NoError(t, dlock.WithMachineLock(func() {
		id := dockerUp("dtest-test-tag", "nginx")

		running := dockerPs()
		assert.Contains(t, running, id)

		dockerKill(id)

		running = dockerPs()
		assert.NotContains(t, running, id)
	}))
}
