package dtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainer(t *testing.T) {
	id := dockerUp("dtest-test-tag", "nginx")

	running := dockerPs()
	assert.Contains(t, running, id)

	dockerKill(id)

	running = dockerPs()
	assert.NotContains(t, running, id)
}
