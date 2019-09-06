package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/teleproxy/pkg/dlock"
	"github.com/datawire/teleproxy/pkg/dtest"
	"github.com/datawire/teleproxy/pkg/k8s"
)

func TestList(t *testing.T) {
	// we get the lock to make sure we are the only thing running
	// because the nat tests interfere with docker functionality
	assert.NoError(t, dlock.WithMachineLock(func() {
		dtest.K8sApply("00-custom-crd.yaml", "custom.yaml")

		c, err := k8s.NewClient(info())
		if err != nil {
			t.Error(err)
			return
		}
		svcs, err := c.List("svc")
		if err != nil {
			t.Error(err)
		}
		found := false
		for _, svc := range svcs {
			if svc.Name() == "kubernetes" {
				found = true
			}
		}
		if !found {
			t.Errorf("did not find kubernetes service")
		}

		customs, err := c.List("customs")
		if err != nil {
			t.Error(err)
		}
		found = false
		for _, cust := range customs {
			if cust.Name() == "xmas" {
				found = true
			}
		}

		if !found {
			t.Errorf("did not find xmas")
		}
	}))
}
