package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"runtime"
	"strings"
	"syscall"

	"github.com/kballard/go-shellquote"
	"github.com/pkg/errors"

	"github.com/datawire/teleproxy/pkg/dlog"
	"github.com/datawire/teleproxy/pkg/logexec"
	"github.com/datawire/teleproxy/pkg/supervisor"
)

// RunAsInfo contains the information required to launch a subprocess as the
// user such that it is likely to function as if the user launched it
// themselves.
type RunAsInfo struct {
	Name string
	Cwd  string
	Env  []string
}

// GetRunAsInfo returns an RAI for the current user context
func GetRunAsInfo() (*RunAsInfo, error) {
	user, err := user.Current()
	if err != nil {
		return nil, errors.Wrap(err, "user.Current()")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "os.Getwd()")
	}
	rai := &RunAsInfo{
		Name: user.Username,
		Cwd:  cwd,
		Env:  os.Environ(),
	}
	return rai, nil
}

// GuessRunAsInfo attempts to construct a RunAsInfo for the user logged in at
// the primary display
func GuessRunAsInfo(p *supervisor.Process) (*RunAsInfo, error) {
	res := RunAsInfo{}
	if runtime.GOOS != "linux" {
		return &res, nil
	}
	pidDirs, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, errors.Wrap(err, "read /proc")
	}
	log := dlog.GetLogger(p.Context())
	for _, fi := range pidDirs {
		if !fi.IsDir() { // Skip /proc files
			continue
		}
		if fi.Sys().(*syscall.Stat_t).Uid == 0 { // Skip root processes
			continue
		}
		// Read the command line for this proc
		cmdline, err := ioutil.ReadFile("/proc/" + fi.Name() + "/cmdline")
		if err != nil {
			log.Printf("Guess/cmdline: Skipping %q: %v", fi.Name(), err)
			continue
		}
		// Skip programs that are not X
		args := bytes.FieldsFunc(cmdline, func(r rune) bool { return r == 0 || r == 32 })
		if len(args) == 0 || !bytes.ContainsRune(args[0], 'X') {
			continue
		}
		log.Printf("Guess: Trying env info from: %q", args[0])
		// Capture the environment for this proc
		environBlob, err := ioutil.ReadFile("/proc/" + fi.Name() + "/environ")
		if err != nil {
			log.Printf("Guess/environ: Skipping %q: %v", fi.Name(), err)
			continue
		}
		environBytes := bytes.Split(environBlob, []byte{0})
		environ := make([]string, len(environBytes))
		display := ""
		for idx := 0; idx < len(environBytes); idx++ {
			entry := string(environBytes[idx])
			environ[idx] = entry
			switch {
			case strings.HasPrefix(entry, "USER="):
				res.Name = entry[5:]
			case strings.HasPrefix(entry, "HOME="):
				res.Cwd = entry[5:]
			case strings.HasPrefix(entry, "DISPLAY="):
				display = entry[8:]
			}
		}
		if len(display) == 0 {
			display = os.Getenv("DISPLAY")
			if len(display) > 0 {
				environ = append(environ, fmt.Sprintf("DISPLAY=%s", display))
			}
		}
		res.Env = environ
		break
	}
	if len(res.Env) == 0 {
		return nil, errors.New("Guess: X server process not found")
	}
	if len(res.Cwd) == 0 || len(res.Name) == 0 {
		return nil, errors.New("Guess: Valid USER/HOME not found")
	}
	return &res, nil
}

// Command returns a logexec.Cmd that is configured to run a subprocess as
// the user in this context.
func (rai *RunAsInfo) Command(ctx context.Context, args ...string) *logexec.Cmd {
	if rai == nil {
		rai = &RunAsInfo{}
	}
	var cmd *logexec.Cmd
	if rai.Name == "root" || len(rai.Name) == 0 {
		cmd = logexec.CommandContext(ctx, args[0], args[1:]...)
	} else {
		if runtime.GOOS == "darwin" {
			// MacOS `su` doesn't appear to propagate signals and
			// `sudo` is always (?) available.
			sudoOpts := []string{"--user", rai.Name, "--set-home", "--preserve-env", "--"}
			cmd = logexec.CommandContext(ctx, "sudo", append(sudoOpts, args...)...)
		} else {
			// FIXME(ark3): The above _should_ work on Linux, but
			// doesn't work on my machine. I don't know why (yet).
			cmd = logexec.CommandContext(ctx, "su", "-m", rai.Name, "-c", shellquote.Join(args...))
		}
	}
	cmd.Env = rai.Env
	cmd.Dir = rai.Cwd
	return cmd
}
