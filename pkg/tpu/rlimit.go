package tpu

import (
	"context"
	"syscall"

	"github.com/datawire/teleproxy/pkg/dlog"
)

func Rlimit(ctx context.Context) {
	log := dlog.GetLogger(ctx)

	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Println("TPY: error getting rlimit:", err)
	} else {
		log.Println("TPY: initial rlimit:", rLimit)
	}

	rLimit.Max = 999999
	rLimit.Cur = 999999
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Println("TPY: Error setting rlimit:", err)
	}

	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Println("TPY: Error getting rlimit:", err)
	} else {
		log.Println("TPY: Final rlimit", rLimit)
	}
}
