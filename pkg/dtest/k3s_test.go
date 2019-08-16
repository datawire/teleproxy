package dtest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// we get the lock to make sure we are the only thing running
	// because the nat tests interfere with docker functionality
	WithMachineLock(func() {
		os.Exit(m.Run())
	})
}

func TestContainer(t *testing.T) {
	id, err := dockerUp("dtest-test-tag", "nginx")
	if err != nil {
		t.Fatal(err)
	}

	running, err := dockerPs()
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, running, id)

	dockerKill(id)

	running, err = dockerPs()
	if err != nil {
		t.Fatal(err)
	}
	assert.NotContains(t, running, id)
}
