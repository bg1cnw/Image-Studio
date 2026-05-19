package client

import (
	"slices"
	"testing"
)

func TestBuildCurlArgsRequiredFlags(t *testing.T) {
	args := BuildCurlArgs("sk-test", "https://gptcodex.top/v1/responses", "request.json")
	if !slices.Contains(args, "-N") {
		t.Errorf("missing -N flag for streaming")
	}
	if !slices.Contains(args, "--data-binary") {
		t.Errorf("missing --data-binary")
	}
	if !slices.Contains(args, "@request.json") {
		t.Errorf("body path arg missing")
	}
	if slices.Contains(args, "-o") {
		t.Errorf("must not redirect to file with -o (streaming to stdout required)")
	}
	// Authorization header must be present and well-formed.
	var foundAuth bool
	for _, a := range args {
		if a == "Authorization: Bearer sk-test" {
			foundAuth = true
		}
	}
	if !foundAuth {
		t.Errorf("Authorization header missing")
	}
}
