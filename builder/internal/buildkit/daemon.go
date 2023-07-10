package buildkit

import (
	"context"
	"io"
	"os/exec"
)

type DaemonOpt func(*Daemon)
type Daemon struct {
	addr   string
	stdout io.Writer
	stderr io.Writer
}

// DaemonOpt: Set an address listen to.
// If the Address is a unix socket, you need to be sure
// buildkit can write to that place as the Daemon
// runs in a rootless environment.
// https://github.com/moby/buildkit/blob/master/docs/rootless.md
func WithAddress(addr string) DaemonOpt {
	return func(d *Daemon) {
		d.addr = addr
	}
}

// DaemonOpt: Set a writer to send the stdout to.
// This can be useful for debugging purposes
// Can be set to nil if previously set.
func WithStdout(writer io.Writer) DaemonOpt {
	return func(d *Daemon) {
		d.stdout = writer
	}
}

// DaemonOpt: Set a writer to send the stderr to.
// Can be set to nil if previously set.
func WithStderr(writer io.Writer) DaemonOpt {
	return func(d *Daemon) {
		d.stderr = writer
	}
}

func NewDaemon(opts ...DaemonOpt) *Daemon {
	daemon := &Daemon{}

	for _, option := range opts {
		option(daemon)
	}

	return daemon
}

func (d *Daemon) Start(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "buildkitd")
	cmd.Stderr = d.stderr
	cmd.Stdout = d.stdout

	if len(d.addr) != 0 {
		cmd.Args = append(cmd.Args, "--addr", d.addr)
	}

	return cmd.Run()
}
