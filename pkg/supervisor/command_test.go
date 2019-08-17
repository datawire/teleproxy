package supervisor

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/datawire/teleproxy/pkg/dlog"
)

func TestMustCapture(t *testing.T) {
	MustRun("bob", func(p *Process) error {
		result := p.Command("echo", "this", "is", "a", "test").MustCapture(nil)
		if result != "this is a test\n" {
			t.Errorf("unexpected result: %v", result)
		}
		return nil
	})
}

func TestCaptureError(t *testing.T) {
	MustRun("bob", func(p *Process) error {
		_, err := p.Command("nosuchcommand").Capture(nil)
		if err == nil {
			t.Errorf("expected an error")
		}
		return nil
	})
}

func TestCaptureExitError(t *testing.T) {
	MustRun("bob", func(p *Process) error {
		_, err := p.Command("test", "1", "==", "0").Capture(nil)
		if err == nil {
			t.Errorf("expected an error")
		}
		return nil
	})
}

func TestCaptureInput(t *testing.T) {
	MustRun("bob", func(p *Process) error {
		output, err := p.Command("cat").Capture(strings.NewReader("hello"))
		if err != nil {
			t.Errorf("unexpected error")
		}
		if output != "hello" {
			t.Errorf("expected hello, got %v", output)
		}
		return nil
	})
}

func TestCommandRun(t *testing.T) {
	MustRun("bob", func(p *Process) error {
		err := p.Command("ls").Run()
		if err != nil {
			t.Errorf("unexpted error: %v", err)
		}
		return nil
	})
}

func TestCommandRunLogging(t *testing.T) {
	logOutput := new(strings.Builder)
	ctx := dlog.WithLogger(context.Background(),
		dlog.WrapLogrus(&logrus.Logger{
			Out: logOutput,
			Formatter: &logrus.TextFormatter{
				DisableTimestamp: true,
			},
			Hooks: make(logrus.LevelHooks),
			Level: logrus.DebugLevel,
		}))

	sup := WithContext(ctx)
	sup.Supervise(&Worker{
		Name: "charles",
		Work: func(p *Process) error {
			// The "cat" in the command is important, otherwise the
			// ordering of the "stdin < EOF" and the "stdout+stderr > 1"
			// lines could go either way.
			return p.Command("bash", "-c", "cat; for i in $(seq 1 3); do echo $i; sleep 0.2; done").Run()
		},
	})
	errs := sup.Run()
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}

	//nolint:lll
	expectedLines := []string{
		`level=info msg="charles: starting"`,
		`level=info msg="[pid:XXPIDXX] started command []string{\"bash\", \"-c\", \"cat; for i in $(seq 1 3); do echo $i; sleep 0.2; done\"}" worker=charles`,
		`level=info msg="[pid:XXPIDXX] stdin  < EOF" worker=charles`,
		`level=info msg="[pid:XXPIDXX] stdout+stderr > \"1\\n\"" worker=charles`,
		`level=info msg="[pid:XXPIDXX] stdout+stderr > \"2\\n\"" worker=charles`,
		`level=info msg="[pid:XXPIDXX] stdout+stderr > \"3\\n\"" worker=charles`,
		`level=info msg="[pid:XXPIDXX] finished successfully: exit status 0" worker=charles`,
		`level=info msg=exited worker=charles`,
		``,
	}

	receivedLines := strings.Split(regexp.MustCompile("pid:[0-9]+").ReplaceAllString(logOutput.String(), "pid:XXPIDXX"), "\n") //nolint:lll
	if len(receivedLines) != len(expectedLines) {
		t.Log("log output didn't have the correct number of lines:")
		t.Logf("expected lines: %d", len(expectedLines))
		for i, line := range expectedLines {
			t.Logf("expected line %d: %q", i, line)
		}
		t.Logf("received lines: %d", len(receivedLines))
		for i, line := range receivedLines {
			t.Logf("received line %d: %q", i, line)
		}
		t.FailNow()
	}
	for i, expectedLine := range expectedLines {
		receivedLine := receivedLines[i]
		if receivedLine != expectedLine {
			t.Errorf("log output line %d didn't match expectations:\n"+
				"expected: %q\n"+
				"received: %q\n",
				i, expectedLine, receivedLine)
		}
	}
}
