package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmCleanupRequiresNamespace(t *testing.T) {
	var output bytes.Buffer
	if confirmCleanup("owner", 3, strings.NewReader("wrong\n"), &output) {
		t.Fatal("confirmCleanup returned true for the wrong confirmation text")
	}

	if !strings.Contains(output.String(), "type: owner") {
		t.Fatalf("confirmation prompt = %q, want namespace prompt", output.String())
	}
}

func TestConfirmCleanupAcceptsExactNamespace(t *testing.T) {
	var output bytes.Buffer
	if !confirmCleanup("owner", 3, strings.NewReader("owner\n"), &output) {
		t.Fatal("confirmCleanup returned false for exact namespace confirmation")
	}
}
