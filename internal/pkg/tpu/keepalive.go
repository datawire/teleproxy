package tpu

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type Keeper struct {
	cancel func()
	done   chan error
}

func (k *Keeper) Shutdown() {
	k.cancel()
	k.Wait()
}

func (k *Keeper) Wait() {
	<-k.done
}

// Keepalive ...
func Keepalive(limit int, input string, program string, args ...string) *Keeper {
	ctx, cancel := context.WithCancel(context.Background())
	k := &Keeper{
		done:   make(chan error),
		cancel: cancel,
	}
	go func() {
		defer close(k.done)
		retry := 0
		for limit == 0 || retry <= limit {
			cmd := exec.CommandContext(ctx, program, args...)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Stdin = bytes.NewBufferString(input)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stdout
			log.Printf("%s", strings.Join(cmd.Args, " "))
			err := cmd.Run()
			if ctx.Err() != nil || err == nil {
				return
			}
			retry++
			log.Printf("restarting...")
			time.Sleep(time.Second)
		}
	}()
	return k
}
