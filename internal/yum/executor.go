package yum

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Executor runs shell commands for testing or production.
type Executor interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) ([]byte, error)
}

// realExecutor executes commands for real.
type realExecutor struct{}

func (e *realExecutor) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w (stderr: %s)", name, args, err, stderr.String())
	}
	return nil
}

func (e *realExecutor) Output(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %v: %w (stderr: %s)", name, args, err, stderr.String())
	}
	return out, nil
}

func (m *YumManager) runCommand(name string, args ...string) error {
	return m.Executor.Run(name, args...)
}

func (m *YumManager) runCommandWithOutput(name string, args ...string) ([]byte, error) {
	return m.Executor.Output(name, args...)
}

func wrapCommandError(err error, op string, subject string) error {
	if subject != "" {
		return fmt.Errorf("%s %q: %w", op, subject, err)
	}
	return fmt.Errorf("%s: %w", op, err)
}
