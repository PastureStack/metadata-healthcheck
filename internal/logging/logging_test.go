package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggersSeparateInformationalAndErrorStreams(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	info, failure := newLoggers(&stdout, &stderr)

	info.Print("configuration unchanged")
	failure.Print("configuration failed")

	if !strings.Contains(stdout.String(), "configuration unchanged") {
		t.Fatalf("informational output was not written to stdout: %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "configuration failed") {
		t.Fatalf("error output leaked into stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "configuration failed") {
		t.Fatalf("error output was not written to stderr: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "configuration unchanged") {
		t.Fatalf("informational output leaked into stderr: %q", stderr.String())
	}
}
