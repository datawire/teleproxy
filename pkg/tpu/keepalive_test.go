package tpu

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestKeepalive(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "lines")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)
	tmpfile := filepath.Join(tmpdir, "lines")

	k := NewKeeper("TST", "echo hi >> "+tmpfile)
	k.Start()
	time.Sleep(3500 * time.Millisecond)
	k.Stop()
	k.Wait()
	dat, err := ioutil.ReadFile(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Count(dat, []byte("\n"))
	if lines != 4 {
		t.Errorf("incorrect number of lines: %v", 4)
	}
}
