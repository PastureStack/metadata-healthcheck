package main

import "testing"

func TestDefaultMetadataURLUsesControlPlaneLinkLocalEndpoint(t *testing.T) {
	const want = "http://169.254.169.250/2015-12-19"
	if defaultMetadataURL != want {
		t.Fatalf("defaultMetadataURL = %q, want %q", defaultMetadataURL, want)
	}
}
