package supervisor

import (
	"context"
	"io"

	"github.com/datawire/teleproxy/pkg/dlog"
	"github.com/datawire/teleproxy/pkg/logexec"
)

// Cmd is a backwards-compatibility shim.
//
// It is a logexec.Cmd with a few extra compatibility methods.
type Cmd struct {
	*logexec.Cmd
}

// Command is a backwards-compatibility shim.
func Command(prefix, name string, args ...string) (result *Cmd) {
	ctx := dlog.WithLoggerField(context.TODO(), "worker", prefix)
	return &Cmd{logexec.CommandContext(ctx, name, args...)}
}

// Command is a backwards-compatibility shim.
//
// Command creates a command that automatically logs inputs, outputs,
// and exit codes to the process logger.
func (p *Process) Command(name string, args ...string) *Cmd {
	ctx := p.Context()
	return &Cmd{logexec.CommandContext(ctx, name, args...)}
}

// Capture is a backwards-compatibility shim.
//
// Capture runs a command with the supplied input and captures the
// output as a string.
func (c *Cmd) Capture(stdin io.Reader) (output string, err error) {
	c.Stdin = stdin
	outputBytes, err := c.Output()
	return string(outputBytes), err
}

// MustCapture is a backwards-compatibility shim.
//
// MustCapture is like Capture, but panics if there is an error.
func (c *Cmd) MustCapture(stdin io.Reader) (output string) {
	output, err := c.Capture(stdin)
	if err != nil {
		panic(err)
	}
	return output
}

// CaptureErr is a backwards-compatibility shim.
//
// CaptureErr runs a command with the supplied input and captures
// stdout and stderr as a string.
func (c *Cmd) CaptureErr(stdin io.Reader) (output string, err error) {
	c.Stdin = stdin
	outputBytes, err := c.CombinedOutput()
	return string(outputBytes), err
}

// MustCaptureErr is a backwards-compatibility shim.
//
// MustCaptureErr is like CaptureErr, but panics if there is an error.
func (c *Cmd) MustCaptureErr(stdin io.Reader) (output string) {
	output, err := c.CaptureErr(stdin)
	if err != nil {
		panic(err)
	}
	return output
}
