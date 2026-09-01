package bootstrapcli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionUsesCapturableOutput(t *testing.T) {
	t.Parallel()
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	if err := Execute(t.Context(), stdout, stderr, []string{"version"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "version=") || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestApplyValidatesRequiredConnectionConfiguration(t *testing.T) {
	t.Parallel()
	command := NewRootCommand(new(bytes.Buffer), new(bytes.Buffer))
	command.SetArgs([]string{"apply"})
	err := command.ExecuteContext(t.Context())
	if err == nil || !strings.Contains(err.Error(), "base-url and authorization are required") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestCompletionRejectsUnknownShellAsUsageError(t *testing.T) {
	t.Parallel()
	command := NewRootCommand(new(bytes.Buffer), new(bytes.Buffer))
	command.SetArgs([]string{"completion", "unknown"})
	if err := command.ExecuteContext(t.Context()); err == nil {
		t.Fatal("completion accepted an unknown shell")
	}
}
