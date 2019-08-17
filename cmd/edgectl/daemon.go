package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"

	"github.com/datawire/teleproxy/pkg/dlog"
	"github.com/datawire/teleproxy/pkg/supervisor"
	"github.com/datawire/teleproxy/pkg/dlog"
)

// Daemon represents the state of the Edge Control Daemon
type Daemon struct {
	network    Resource
	cluster    *KCluster
	bridge     Resource
	trafficMgr *TrafficManager
	intercepts []*Intercept
}

// RunAsDaemon is the main function when executing as the daemon
func RunAsDaemon() error {
	if os.Geteuid() != 0 {
		return errors.New("edgectl daemon must run as root")
	}

	d := &Daemon{}

	logger := SetUpLogging()
	ctx := dlog.WithLogger(context.Background(), logger)

	sup := supervisor.WithContext(ctx)
	sup.Supervise(&supervisor.Worker{
		Name: "daemon",
		Work: d.acceptLoop,
	})
	sup.Supervise(&supervisor.Worker{
		Name:     "signal",
		Requires: []string{"daemon"},
		Work:     WaitForSignal,
	})
	sup.Supervise(&supervisor.Worker{
		Name:     "setup",
		Requires: []string{"daemon"},
		Work: func(p *supervisor.Process) error {
			if err := d.MakeNetOverride(p); err != nil {
				return err
			}
			p.Ready()
			return nil
		},
	})

	logger.Printf("---")
	logger.Printf("Edge Control daemon %s starting...", displayVersion)
	logger.Printf("PID is %d", os.Getpid())
	runErrors := sup.Run()

	logger.Printf("")
	if len(runErrors) > 0 {
		logger.Printf("Daemon has exited with %d error(s):", len(runErrors))
		for _, err := range runErrors {
			logger.Printf("- %v", err)
		}
	}
	logger.Printf("Edge Control daemon %s is done.", displayVersion)
	return errors.New("edgectl daemon has exited")
}

func (d *Daemon) acceptLoop(p *supervisor.Process) error {
	// Listen on unix domain socket
	unixListener, err := net.Listen("unix", socketName)
	if err != nil {
		return errors.Wrap(err, "chmod")
	}
	err = os.Chmod(socketName, 0777)
	if err != nil {
		return errors.Wrap(err, "chmod")
	}

	p.Ready()
	Notify(p, "Running")
	defer Notify(p, "Terminated")

	return p.DoClean(
		func() error {
			for {
				conn, err := unixListener.Accept()
				if err != nil {
					return errors.Wrap(err, "accept")
				}
				_ = p.Go(func(p *supervisor.Process) error {
					return d.handle(p, conn)
				})
			}
		},
		unixListener.Close,
	)
}

func (d *Daemon) handle(p *supervisor.Process, conn net.Conn) error {
	defer conn.Close()
	log := dlog.GetLogger(p.Context())

	decoder := json.NewDecoder(conn)
	data := &ClientMessage{}
	if err := decoder.Decode(data); err != nil {
		log.Printf("Failed to read message: %v", err)
		fmt.Fprintln(conn, "API mismatch. Server", displayVersion)
		return nil
	}
	if data.APIVersion != apiVersion {
		log.Printf("API version mismatch (got %d, need %d)", data.APIVersion, apiVersion)
		fmt.Fprintf(conn, "API version mismatch (got %d, server %s)", data.APIVersion, displayVersion)
		return nil
	}
	log.Printf("Received command: %q", data.Args)

	err := d.handleCommand(p, conn, data)
	if err != nil {
		log.Printf("Command processing failed: %v", err)
	}

	log.Print("Done")
	return nil
}
