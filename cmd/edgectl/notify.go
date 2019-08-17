package main

import (
	"fmt"
	"runtime"

	"github.com/datawire/teleproxy/pkg/supervisor"
	"github.com/datawire/teleproxy/pkg/dlog"
)

var notifyRAI *RunAsInfo

// Notify displays a desktop banner notification to the user
func Notify(p *supervisor.Process, message string) {
	log := dlog.GetLogger(p.Context())
	if notifyRAI == nil {
		var err error
		notifyRAI, err = GuessRunAsInfo(p)
		if err != nil {
			log.Print(err)
			notifyRAI = &RunAsInfo{}
		}
	}

	var args []string
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf("display notification \"Edge Control Daemon\" with title \"%s\"", message)
		args = []string{"osascript", "-e", script}
	case "linux":
		args = []string{"notify-send", "Edge Control Daemon", message}
	default:
		return
	}

	log.Printf("NOTIFY: %s", message)
	cmd := notifyRAI.Command(p.Context(), args...)
	if err := cmd.Run(); err != nil {
		log.Printf("ERROR while notifying: %v", err)
	}
}

// MaybeNotify displays a notification only if a value changes
func MaybeNotify(p *supervisor.Process, name string, old, new bool) {
	if old != new {
		Notify(p, fmt.Sprintf("%s: %t -> %t", name, old, new))
	}
}
