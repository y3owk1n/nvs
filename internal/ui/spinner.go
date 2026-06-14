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
	"github.com/sirupsen/logrus"
)

// SafeSpinner wraps *spinner.Spinner and synchronizes updates to its Suffix
// using the spinner's own internal mutex. The spinner reads Suffix on its
// animation goroutine under the same mutex (see briandowns/spinner spin
// loop), so all external writes must be serialized through Lock/Unlock to
// avoid data races.
type SafeSpinner struct {
	sp *spinner.Spinner
}

// NewSafeSpinner returns a SafeSpinner backed by the given spinner.
func NewSafeSpinner(sp *spinner.Spinner) *SafeSpinner {
	return &SafeSpinner{sp: sp}
}

// Start begins the spinner animation.
func (s *SafeSpinner) Start() {
	s.sp.Start()
}

// Stop ends the spinner animation. Safe to call multiple times.
func (s *SafeSpinner) Stop() {
	s.sp.Stop()
}

// SetSuffix updates the spinner's suffix text in a race-free manner.
func (s *SafeSpinner) SetSuffix(suffix string) {
	s.sp.Lock()
	s.sp.Suffix = suffix
	s.sp.Unlock()
}

// SetPrefix updates the spinner's prefix text in a race-free manner.
func (s *SafeSpinner) SetPrefix(prefix string) {
	s.sp.Lock()
	s.sp.Prefix = prefix
	s.sp.Unlock()
}

// RunCommandWithSpinner executes the provided command with an active spinner that updates its suffix
// based on the command's output. It captures both stdout and stderr and returns an error if the command fails.
func RunCommandWithSpinner(ctx context.Context, spinner *spinner.Spinner, cmd *exec.Cmd) error {
	const goroutineNum = 2

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

		err := scanner.Err()
		if err != nil {
			logrus.Debugf("scanner error reading pipe: %v", err)
		}
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(goroutineNum)

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
		// Kill the process to ensure pipes close and goroutines can exit
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		// Wait for goroutines to finish reading
		waitGroup.Wait()

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
