package k3sctl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/teleproxy/pkg/dlock"
)

func TestMain(m *testing.M) {
	// we get the lock to make sure we are the only thing running
	// because the nat tests interfere with docker functionality
	dlock.WithMachineLock(func() {
		os.Exit(m.Run())
	})
}

func TestContainer(t *testing.T) {
	id := dockerUp("dtest-test-tag", "nginx")

	running := dockerPs()
	assert.Contains(t, running, id)

	dockerKill(id)

	running = dockerPs()
	assert.NotContains(t, running, id)
}
