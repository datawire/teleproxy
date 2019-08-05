package dtest

import (
	"fmt"
	"os"
	"syscall"
)

const pattern = "/tmp/datawire-machine-scoped-%s.lock"

func exit(filename string, err error) {
	fmt.Fprintf(os.Stderr, "error trying to acquire lock on %s: %v\n", filename, err)
	os.Exit(1)
}

// WithMachineLock executes the supplied body with a guarantee that it
// is the only code running (via WithMachineLock) on the machine.
func WithMachineLock(body func()) {
	WithNamedMachineLock("default", body)
}

func WithNamedMachineLock(name string, body func()) {
	filename := fmt.Sprintf(pattern, name)
	var file *os.File
	var err error
	func() {
		old := syscall.Umask(0)
		defer syscall.Umask(old)
		/* #nosec */
		file, err = os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	}()

	if err != nil {
		exit(filename, err)
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
	if err != nil {
		err = &os.PathError{Op: "flock", Path: file.Name(), Err: err}
		exit(filename, err)
	}
	defer func() {
		file.Close()
	}()

	body()
}
