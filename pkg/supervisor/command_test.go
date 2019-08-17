package supervisor

import (
	"context"
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
			cmd := p.Command("bash", "-c", "for i in $(seq 1 3); do echo $i; sleep 0.2; done")
			if err := cmd.Run(); err != nil {
				t.Errorf("unexpted error: %v", err)
			}
			logOutputLines := strings.Split(strings.TrimSuffix(logOutput.String(), "\n"), "\n")
			if len(logOutputLines) != 6 {
				t.Log("Expected 6 lines: process start, cmd start, 1, 2, 3, cmd end")
				t.Logf("Got (%d lines): %q", len(logOutputLines), logOutputLines)
				t.Fail()
			}
			return nil
		},
	})
	sup.Run()
}
