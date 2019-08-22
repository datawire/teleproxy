package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"strings"
	"time"

	"github.com/datawire/teleproxy/pkg/dtest"
	"github.com/datawire/teleproxy/pkg/dtest/testprocess"
)

// Assert has convenient functions for doing test assertions.
type Assert struct {
	T testing.TB
}

func (a *Assert) HasError(err error, msg string) {
	a.T.Helper()
	if err == nil {
		a.T.Fatalf("%s: didn't get expected error", msg)
	}
}

func (a *Assert) NotError(err error, msg string) {
	a.T.Helper()
	if err != nil {
		a.T.Fatalf("%s: unexpected error: %v", msg, err)
	}
}

var kubeconfig string

func TestMain(m *testing.M) {
	testprocess.Dispatch()
	kubeconfig = dtest.Kubeconfig()
	os.Setenv("DTEST_KUBECONFIG", kubeconfig)
	dtest.WithMachineLock(func() {
		os.Exit(m.Run())
	})
}

func showArgs(args []string) {
	fmt.Print("+")
	for _, arg := range args {
		fmt.Print(" ", arg)
	}
	fmt.Println()
}

func run(args ...string) error {
	showArgs(args)
	cmd := exec.Command(args[0], args[1:]...)
	return runCmd(cmd)
}

func runCmd(cmd *exec.Cmd) error {
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("==>", err)
	}
	return err
}

// nolint deadcode
func capture(args ...string) (string, error) {
	showArgs(args)
	cmd := exec.Command(args[0], args[1:]...)
	return captureCmd(cmd)
}

func captureCmd(cmd *exec.Cmd) (string, error) {
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	cmd.Stdout = nil
	cmd.Stderr = nil
	outBytes, err := cmd.CombinedOutput()
	out := string(outBytes)
	if len(out) > 0 {
		fmt.Print(out)
		if out[len(out)-1] != '\n' {
			fmt.Println(" [no newline]")
		}
	}
	if err != nil {
		fmt.Println("==>", err)
	}
	return out, err
}

func runMe(args ...string) {
	os.Args = append([]string{"edgectl"}, args...)
	showArgs(os.Args)
	main()
}

func runMeVersion()    { runMe("version") }
func runMeStatus()     { runMe("status") }
func runMeConnect()    { runMe("connect") }
func runMeDisconnect() { runMe("disconnect") }
func runMeDaemon()     { runMe("daemon") }
func runMeQuit()       { runMe("quit") }

// doBuildExecutable calls make in a subprocess running as the user
func doBuildExecutable() {
	if !strings.Contains(os.Getenv("MAKEFLAGS"), "--jobserver-auth") {
		err := run("make", "-C", "../..", "bin_"+runtime.GOOS+"_"+runtime.GOARCH+"/edgectl")
		if err != nil {
			log.Fatalf("build executable: %v", err)
		}
	}
}

var eVersion = testprocess.Make(runMeVersion)
var eStatus = testprocess.Make(runMeStatus)
var eConnect = testprocess.Make(runMeConnect)
var eDisconnect = testprocess.Make(runMeDisconnect)
var eDaemon = testprocess.Make(runMeDaemon)
var eQuit = testprocess.Make(runMeQuit)
var buildExecutable = testprocess.Make(doBuildExecutable)

var executable = "../../bin_" + runtime.GOOS + "_" + runtime.GOARCH + "/edgectl"

func TestSmokeOutbound(t *testing.T) {
	assert := Assert{t}
	var out string
	var err error

	// Setup
	assert.NotError(run("sudo", "true"), "acquire privileges")
	assert.NotError(run("printenv", "KUBECONFIG"), "ensure cluster is set")
	assert.NotError(run("sudo", "rm", "-f", "/tmp/edgectl.log"), "remove old log")

	// Cluster setup
	assert.NotError(
		run("kubectl", "delete", "pod", "teleproxy", "--ignore-not-found", "--wait=true"),
		"check cluster connectivity",
	)
	namespace := fmt.Sprintf("edgectl-%d", os.Getpid())
	nsArg := fmt.Sprintf("--namespace=%s", namespace)
	assert.NotError(run("kubectl", "create", "namespace", namespace), "create test namespace")
	defer func() {
		assert.NotError(
			run("kubectl", "delete", "namespace", namespace, "--wait=false"),
			"delete test namespace",
		)
	}()
	assert.NotError(
		run("kubectl", nsArg, "create", "deploy", "hello-world", "--image=ark3/hello-world"),
		"create deployment",
	)
	assert.NotError(
		run("kubectl", nsArg, "expose", "deploy", "hello-world", "--port=80", "--target-port=8000"),
		"create service",
	)
	assert.NotError(
		run("kubectl", nsArg, "get", "svc,deploy", "hello-world"),
		"check svc/deploy",
	)

	// Pre-daemon tests
	assert.HasError(runCmd(eStatus), "status with no daemon")
	assert.HasError(runCmd(eDaemon), "daemon without sudo")

	// Daemon tests
	assert.NotError(runCmd(buildExecutable), "build executable")
	assert.NotError(run("sudo", executable, "daemon"), "launch daemon")
	defer func() { assert.NotError(runCmd(eQuit), "quit daemon") }()
	assert.NotError(runCmd(eVersion), "version with daemon")
	eStatus = testprocess.Make(runMeStatus)
	assert.NotError(runCmd(eStatus), "status with daemon")

	// Wait for network overrides
	func() {
		for i := 0; i < 30; i++ {
			eStatus = testprocess.Make(runMeStatus)
			out, _ := captureCmd(eStatus)
			if !strings.Contains(out, "Network overrides NOT established") {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		t.Fatal("Net overrides timeout")
	}()

	assert.NotError(runCmd(eConnect), "connect")
	defer func() {
		assert.NotError(
			run("kubectl", "delete", "pod", "teleproxy", "--ignore-not-found", "--wait=false"),
			"make next time quicker",
		)
	}()
	eStatus = testprocess.Make(runMeStatus)
	out, err = captureCmd(eStatus)
	assert.NotError(err, "status connected")
	if !strings.Contains(out, "Context") {
		t.Fatal("Expected Context in status output")
	}

	// Wait for bridge
	func() {
		for i := 0; i < 30; i++ {
			eStatus = testprocess.Make(runMeStatus)
			out, _ := captureCmd(eStatus)
			if strings.Contains(out, "Proxy:         ON") {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		t.Fatal("timed out waiting for net overrides")
	}()

	// Bridge tests
	assert.NotError(run("curl", "-sv", "hello-world."+namespace), "check bridge")

	// Wind down
	eStatus = testprocess.Make(runMeStatus)
	out, err = captureCmd(eStatus)
	assert.NotError(err, "status connected")
	if !strings.Contains(out, "Context") {
		t.Fatal("Expected Context in status output")
	}
	assert.NotError(runCmd(eDisconnect), "disconnect")
	eStatus = testprocess.Make(runMeStatus)
	out, err = captureCmd(eStatus)
	assert.NotError(err, "status disconnected")
	if !strings.Contains(out, "Not connected") {
		t.Fatal("Expected Not connected in status output")
	}
	assert.HasError(run("curl", "-sv", "hello-world."+namespace), "check disconnected")
}
