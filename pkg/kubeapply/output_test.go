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
	Action     func(*testing.T, *kubeapply.StatusWriter)
	Appearance string
}

func iocheck(t *testing.T, nExp int, fn func() (int, error)) {
	t.Helper()
	nAct, e := fn()
	if nAct != nExp {
		t.Errorf("io: expected %d bytes, got %d", nExp, nAct)
	}
	if e != nil {
		t.Errorf("io: unexpected error: %v", e)
	}
}

func errcheck(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Errorf("io: unexpected error: %v", e)
	}
}

func TestStatusWriter(t *testing.T) {
	testcases := map[string][]statusWriterTestStep{
		"simple output, trailing newline": {
			{
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					iocheck(t, 4, func() (int, error) { return fmt.Fprintln(w, "foo") })
				},
				Appearance: "" +
					"foo\n" +
					"----\n" +
					"",
			},
		},
		"simple output, no trailing newline": {
			{
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					iocheck(t, 3, func() (int, error) { return fmt.Fprintf(w, "foo") })
				},
				Appearance: "" +
					"foo\n" +
					"----\n" +
					"",
			},
		},
		"no output, but a status": {
			{
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					errcheck(t, w.SetStatus("A", "B"))
				},
				Appearance: "" +
					"----\n" +
					"A: B\n" +
					"",
			},
		},
		"moderately complex example": {
			{
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					iocheck(t, 7, func() (int, error) { return fmt.Fprintln(w, "line 1") })
				},
				Appearance: "" +
					"line 1\n" +
					"----\n" +
					"",
			},
			{
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					errcheck(t, w.SetStatus("Zoom", "Zombocom"))
					errcheck(t, w.SetStatus("Bogart", "waiting"))
				},
				Appearance: "" +
					"line 1\n" +
					"----------------\n" +
					"Bogart: waiting\n" +
					"Zoom  : Zombocom\n" +
					"",
			},
			{
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					iocheck(t, 6, func() (int, error) { return fmt.Fprintf(w, "line 2") })
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
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					errcheck(t, w.SetStatus("Zoom", ":("))
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
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					iocheck(t, 16, func() (int, error) { return fmt.Fprintln(w, " more of line 2") })
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
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					iocheck(t, 7, func() (int, error) { return fmt.Fprintln(w, "line 3") })
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
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					iocheck(t, 7, func() (int, error) { return fmt.Fprintln(w, "line 4") })
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
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					errcheck(t, w.SetStatus("Bogart", "done"))
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
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					iocheck(t, 14, func() (int, error) { return fmt.Fprintln(w, "line 5\nline 6") })
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
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					errcheck(t, w.SetStatus("A", "B"))
				},
				Appearance: "" +
					"----\n" +
					"A: B\n" +
					"",
			},
			{
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					iocheck(t, 6, func() (int, error) { return fmt.Fprintf(w, "foobar") })
				},
				Appearance: "" +
					"foobar\n" +
					"----\n" +
					"A: B\n" +
					"",
			},
			{
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					iocheck(t, 9, func() (int, error) { return fmt.Fprintln(w, "\rFOO\nbaz") })
				},
				Appearance: "" +
					"FOObar\n" +
					"baz\n" +
					"----\n" +
					"A: B\n" +
					"",
			},
		},
		"short status": {
			{
				Action: func(t *testing.T, w *kubeapply.StatusWriter) {
					errcheck(t, w.SetStatus("A", ""))
				},
				Appearance: "" +
					"----\n" +
					"A: \n" +
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
				step.Action(t, w)
				output, err := exec.Command("emacs", "--script", "testdata/term-render.el", file.Name()).Output()
				if err != nil {
					t.Fatalf("step %d: term-render: %v", i, err)
				}
				if string(output) != step.Appearance {
					raw, _ := ioutil.ReadFile(file.Name())
					t.Errorf("step %d: output rendered incorrectly\n"+
						"Raw: %q\n"+
						"Expected: %q\n"+
						"Received: %q\n",
						i, string(raw), step.Appearance, string(output))
				}
			}
		})
	}
}
