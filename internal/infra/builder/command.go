package builder

import (
	"context"
	"io"
	"os/exec"
)

// defaultExecCommand is the default command execution function.
func defaultExecCommand(ctx context.Context, name string, args ...string) Commander {
	return &execCommand{cmd: exec.CommandContext(ctx, name, args...)}
}

// execCommand wraps exec.Cmd to implement Commander interface.
type execCommand struct {
	cmd *exec.Cmd
}

func (e *execCommand) Run() error {
	return e.cmd.Run()
}

func (e *execCommand) SetDir(dir string) {
	e.cmd.Dir = dir
}

func (e *execCommand) SetStdout(stdout any) {
	// Only io.Writer types are applied; other types are silently ignored
	// to allow test doubles to pass alternative implementations.
	if w, ok := stdout.(io.Writer); ok {
		e.cmd.Stdout = w
	}
}

func (e *execCommand) SetStderr(stderr any) {
	// Only io.Writer types are applied; other types are silently ignored
	// to allow test doubles to pass alternative implementations.
	if w, ok := stderr.(io.Writer); ok {
		e.cmd.Stderr = w
	}
}

func (e *execCommand) StdoutPipe() (any, error) {
	return e.cmd.StdoutPipe()
}

func (e *execCommand) StderrPipe() (any, error) {
	return e.cmd.StderrPipe()
}
