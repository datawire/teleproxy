package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/datawire/teleproxy/pkg/dlog"
	"github.com/datawire/teleproxy/pkg/supervisor"
)

// WaitForSignal is a Worker that calls Shutdown if SIGINT or SIGTERM
// is received.
func WaitForSignal(p *supervisor.Process) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	p.Ready()

	log := dlog.GetLogger(p.Context())
	select {
	case killSignal := <-interrupt:
		switch killSignal {
		case os.Interrupt:
			log.Print("Got SIGINT...")
		case syscall.SIGTERM:
			log.Print("Got SIGTERM...")
		}
		p.Supervisor().Shutdown()
	case <-p.Shutdown():
	}
	return nil
}
