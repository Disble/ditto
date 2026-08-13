package testingtlaboratory

import (
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakelaboratory"
	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/dittotesting/faketestingt"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/stretchr/testify/assert"
)

// TestMarksTheSubtestParallelOnlyWhenAsked watches the call rather than its
// effect.
//
// The effect is unobservable from a test: T.Parallel returns without recording
// anything when its parent has no barrier, which is exactly the shape of a
// hand-built *testing.T, and filling that in makes it block forever sending on
// a nil signal channel. Asserting through the seam keeps this test true across
// Go releases instead of coupling it to whichever private field is current.
func TestMarksTheSubtestParallelOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		parallel bool
		want     int
	}{
		{name: "left serial when not asked", parallel: false, want: 0},
		{name: "marked parallel when asked", parallel: true, want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			fakeT := faketestingt.New()

			laboratory := New(fakeT, fakelaboratory.NewAlways(result.Ok("mutant killed")), test.parallel)
			laboratory.goParallel = func(*testing.T) { calls++ }

			laboratory.Test(
				fakerepository.New(fakerepository.FS{}),
				gomutatedfile.New("test-infection", "some-path.go", nil, nil),
			)
			fakeT.GetSubtest("some-path.go → test-infection").Run()

			assert.Equal(t, test.want, calls)
		})
	}
}
