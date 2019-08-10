package kubeapply_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/datawire/teleproxy/pkg/kubeapply"
)

type statusWriterTestStep struct {
	Action     func(*kubeapply.StatusWriter)
	Appearance string
}

func runTestcase(t *testing.T, testcase []statusWriterTestStep) {
	t.Helper()
}

func TestStatusWriter(t *testing.T) {
	testcases := map[string][]statusWriterTestStep{
		"simple output, trailing newline": {
			{
				Action: func(w *kubeapply.StatusWriter) {
					fmt.Fprintln(w, "foo")
				},
				Appearance: "" +
					"foo\n" +
					"",
			},
		},
		"simple output, no trailing newline": {
			{
				Action: func(w *kubeapply.StatusWriter) {
					fmt.Fprintf(w, "foo")
				},
				Appearance: "" +
					"foo\n" +
					"",
			},
		},
		"no output, but a status": {
			{
				Action: func(w *kubeapply.StatusWriter) {
					w.SetStatus("A", "B")
				},
				Appearance: "" +
					"----\n" +
					"A: B\n" +
					"",
			},
		},
		"moderately complex example": {
			{
				Action: func(w *kubeapply.StatusWriter) {
					fmt.Fprintln(w, "line 1")
				},
				Appearance: "" +
					"line 1\n" +
					"",
			},
			{
				Action: func(w *kubeapply.StatusWriter) {
					w.SetStatus("Zoom", "Zombocom")
					w.SetStatus("Bogart", "waiting")
				},
				Appearance: "" +
					"line 1\n" +
					"----------------\n" +
					"Bogart: waiting\n" +
					"Zoom  : Zombocom\n" +
					"",
			},
			{
				Action: func(w *kubeapply.StatusWriter) {
					fmt.Fprintf(w, "line 2")
				},
				Appearance: "" +
					"line 1\n" +
					"line 2\n" +
					"----------------\n" +
					"Bogart: waiting\n" +
					"Zoom  : Zombocom\n" +
					"",
			},
			{
				Action: func(w *kubeapply.StatusWriter) {
					w.SetStatus("Zoom", ":(")
				},
				Appearance: "" +
					"line 1\n" +
					"line 2\n" +
					"---------------\n" +
					"Bogart: waiting\n" +
					"Zoom  : :(\n" +
					"",
			},
			{
				Action: func(w *kubeapply.StatusWriter) {
					fmt.Fprintln(w, " more of line 2")
				},
				Appearance: "" +
					"line 1\n" +
					"line 2 more of line 2\n" +
					"---------------\n" +
					"Bogart: waiting\n" +
					"Zoom  : :(\n" +
					"",
			},
			{
				Action: func(w *kubeapply.StatusWriter) {
					fmt.Fprintln(w, "line 3")
				},
				Appearance: "" +
					"line 1\n" +
					"line 2 more of line 2\n" +
					"line 3\n" +
					"---------------\n" +
					"Bogart: waiting\n" +
					"Zoom  : :(\n" +
					"",
			},
			{
				Action: func(w *kubeapply.StatusWriter) {
					fmt.Fprintln(w, "line 4")
				},
				Appearance: "" +
					"line 1\n" +
					"line 2 more of line 2\n" +
					"line 3\n" +
					"line 4\n" +
					"---------------\n" +
					"Bogart: waiting\n" +
					"Zoom  : :(\n" +
					"",
			},
			{
				Action: func(w *kubeapply.StatusWriter) {
					w.SetStatus("Bogart", "done")
				},
				Appearance: "" +
					"line 1\n" +
					"line 2 more of line 2\n" +
					"line 3\n" +
					"line 4\n" +
					"------------\n" +
					"Bogart: done\n" +
					"Zoom  : :(\n" +
					"",
			},
			{
				Action: func(w *kubeapply.StatusWriter) {
					fmt.Fprintln(w, "line 5\nline 6")
				},
				Appearance: "" +
					"line 1\n" +
					"line 2 more of line 2\n" +
					"line 3\n" +
					"line 4\n" +
					"line 5\n" +
					"line 6\n" +
					"------------\n" +
					"Bogart: done\n" +
					"Zoom  : :(\n" +
					"",
			},
		},
		"carriage returns partially overwriting a line": {
			{
				Action: func(w *kubeapply.StatusWriter) {
					w.SetStatus("A", "B")
				},
				Appearance: "" +
					"----\n" +
					"A: B\n" +
					"",
			},
			{
				Action: func(w *kubeapply.StatusWriter) {
					fmt.Fprintf(w, "foobar")
				},
				Appearance: "" +
					"foobar\n" +
					"----\n" +
					"A: B\n" +
					"",
			},
			{
				Action: func(w *kubeapply.StatusWriter) {
					fmt.Fprintln(w, "\rFOO\nbaz")
				},
				Appearance: "" +
					"FOObar\n" +
					"baz\n" +
					"----\n" +
					"A: B\n" +
					"",
			},
		},
	}
	for tcName, tcData := range testcases {
		testcase := tcData // capture range variable
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()
			file, err := ioutil.TempFile("", "term-render.")
			if err != nil {
				t.Fatalf("ioutil.TempFile: %v", err)
			}
			defer func() {
				_ = os.Remove(file.Name())
				_ = file.Close()
			}()

			w := kubeapply.NewStatusWriter(file)
			for i, step := range testcase {
				step.Action(w)
				output, err := exec.Command("emacs", "--script", "testdata/term-render.el", file.Name()).Output()
				if err != nil {
					t.Fatalf("step %d: term-render: %v", i, err)
				}
				if string(output) != step.Appearance {
					t.Errorf("step %d: output rendered incorrectly\n"+
						"Expected: %q\n"+
						"Received: %q\n",
						i, step.Appearance, string(output))
				}
			}
		})
	}
}
