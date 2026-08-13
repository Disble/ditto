package faketestingt

import "testing"

// SubTest captures a subtest body so a test can run it on demand.
//
// It deliberately knows nothing about testing.T's internals. An earlier
// version reached into unexported fields with reflect and unsafe to observe
// whether T.Parallel had been called; that observation is not available to any
// caller, and the private layout it depended on changes between Go releases.
// The laboratory exposes a seam for that instead.
type SubTest struct {
	subtest func(*testing.T)
	t       *testing.T
}

func NewSubTest(subtest func(*testing.T)) *SubTest {
	return &SubTest{
		subtest: subtest,
		t:       new(testing.T),
	}
}

func (s *SubTest) Run() {
	s.subtest(s.t)
}

func (s *SubTest) Failed() bool {
	return s.t.Failed()
}
