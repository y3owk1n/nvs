// Package ui provides user interface utilities.
// Package ui provides user interface utilities.
package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/briandowns/spinner"
)

const (
	// GoroutineNum is the number of goroutines used for spinner updates.
	GoroutineNum = 2
)

// RunCommandWithSpinner executes the provided command with an active spinner that updates its suffix
// based on the command's output. It captures both stdout and stderr and returns an error if the command fails.
func RunCommandWithSpinner(ctx context.Context, spinner *spinner.Spinner, cmd *exec.Cmd) error {
	var err error

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	var suffixMutex sync.Mutex

	// updateSpinner reads from the given pipe and updates the spinner's suffix based on the output.
	updateSpinner := func(pipeOutput io.Reader, waitGroup *sync.WaitGroup) {
		defer waitGroup.Done()

		scanner := bufio.NewScanner(pipeOutput)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				suffixMutex.Lock()

				spinner.Suffix = " " + line

				suffixMutex.Unlock()
			}
		}
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(GoroutineNum)

	go updateSpinner(stdoutPipe, &waitGroup)
	go updateSpinner(stderrPipe, &waitGroup)

	// Channel to capture command completion.
	cmdErrChan := make(chan error, 1)
	go func() {
		cmdErrChan <- cmd.Wait()
	}()

	// Wait for either the command to finish or the context to be canceled.
	select {
	case <-ctx.Done():
		return fmt.Errorf("command canceled: %w", ctx.Err())
	case err := <-cmdErrChan:
		// Wait for spinner update routines to finish.
		waitGroup.Wait()

		if err != nil {
			return err
		}
	}

	return nil
}
