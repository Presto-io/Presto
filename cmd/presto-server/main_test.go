package main

import "testing"

func TestShouldInjectAPIKeyHonorsEnvironmentOverride(t *testing.T) {
	t.Setenv("PRESTO_INJECT_API_KEY", "true")
	if !shouldInjectAPIKey("0.0.0.0") {
		t.Fatal("expected PRESTO_INJECT_API_KEY=true to enable injection")
	}

	t.Setenv("PRESTO_INJECT_API_KEY", "false")
	if shouldInjectAPIKey("127.0.0.1") {
		t.Fatal("expected PRESTO_INJECT_API_KEY=false to disable injection")
	}
}
