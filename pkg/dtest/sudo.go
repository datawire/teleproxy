package dtest

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Sudo is intended for use in a TestMain. It will relaunch the test
// executable via sudo if it isn't already running with an effective
// userid of root.
func Sudo() {
	/* #nosec */
	if os.Geteuid() != 0 {
		cmd := exec.Command("sudo", append([]string{"-E"}, os.Args...)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("error re-invoking tests with sudo: %v\n", err)
		}
		os.Exit(cmd.ProcessState.ExitCode())
	}
}

var (
	reportPass = "--- PASS: "
	reportFail = "--- FAIL: "
	reportSkip = "--- SKIP: "
)

func parseReportLine(line, testName string) string {
	for strings.HasPrefix(line, "    ") {
		line = line[4:]
	}
	reportTypes := []string{
		reportPass,
		reportFail,
		reportSkip,
	}
	for _, reportType := range reportTypes {
		if strings.HasPrefix(line, reportType+testName+" (") {
			return reportType
		}
	}
	return ""
}

// RunAsRoot uses sudo to run individual test-cases as root.  This should be the only thing in the
// top-level of the test.  It is not valid to use RunAsRoot for a sub-test (i.e. something run with
// t.Run()).
//
// 	func TestThatNeedsRoot(t *testing.T) {
// 		// There should be no code before RunAsRoot
// 		dtest.RunAsRoot(t, func(t *testing.T) {
// 			// Anything that you would normally put in a test, including sub-tests, may
// 			// go here.  It will run as root instead of as the normal user.
// 			t.Logf("I am %v\n", os.Getuid())
// 		})
// 		// There should be no code after RunAsRoot
// 	}
func RunAsRoot(t *testing.T, f func(t *testing.T)) {
	t.Helper()
	if strings.Contains(t.Name(), "/") {
		t.Fatalf("it is invalid to use RunAsRoot in a sub-test")
	}
	if os.Getuid() != 0 {
		args := []string{"sudo", "--preserve-env", "--"}
		args = append(args, os.Args...)
		args = append(args, "-test.run="+t.Name())

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		// We're going to filter stdout a bit, but mostly just pass it through.
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		defer stdout.Close()

		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}

		// Copy stdout through, but filter a few things:
		//
		//  1. Filter out everything up to and including the "=== RUN TestName\n" line that
		//     indicates our specific test has started.
		//
		//  2. Filter out the final "PASS\n" and "FAIL\n" lines.
		//
		//  3. Filter out the "--- PASS: TestName (0s)\n" line report line for this specific
		//     test, since the parent process will print it again.
		buf := bufio.NewReader(stdout)
		thisTestHasStarted := false
		var status string
		for err == nil {
			var line string
			line, err = buf.ReadString('\n')
			if line == "=== RUN   "+t.Name()+"\n" {
				thisTestHasStarted = true
				continue
			}
			if line == "PASS\n" || line == "FAIL\n" {
				continue
			}
			if reportStatus := parseReportLine(line, t.Name()); reportStatus != "" {
				status = reportStatus
				continue
			}
			if thisTestHasStarted {
				_, _ = io.WriteString(os.Stdout, line)
			}
		}
		if err != io.EOF {
			t.Fatal(err)
		}

		if err := cmd.Wait(); err != nil {
			if eerr, isExit := err.(*exec.ExitError); !isExit || eerr.ExitCode() != 1 || status != reportFail {
				t.Fatal(err)
			}
		}

		switch status {
		case reportSkip:
			t.Skip()
		case reportFail:
			t.Fail()
		}
	} else {
		f(t)
	}
}
