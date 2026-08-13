//go:build mutation

package ditto_test

import (
	"testing"

	"github.com/Disble/ditto"
)

func TestMutation(t *testing.T) {
	ditto.Release(
		t,
		ditto.ForceColors(),
		ditto.WithRepositoryRoot("."),
		ditto.WithTestCommand("make test.failfast MAKEFLAGS="),
		ditto.WithMinimumThreshold(0.5),
		ditto.Parallel(),
		ditto.IgnoreSourceFiles("(^release\\.go$|testdata\\/.*)"),
	)
}
