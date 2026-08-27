package ditto

// RefusalError is the one thing a release says by stopping.
//
// A laboratory that finds the suite already red cannot return that finding:
// Test hands back a future of a result, and every value in it means something
// about a mutant. There is no mutant here — the run has not started, and
// scoring anything would be scoring a suite that tested nothing. So it panics,
// and the panic is the verdict rather than a defect.
//
// It is a type rather than a string because the two entry points need different
// things from it. Release lets it through, so a test fails with the message
// printed as the panic value. Run has to turn it into an error, and recognising
// it by matching text would tie the entry point to a sentence somebody will
// reword.
type RefusalError struct {
	message string
}

// NewRefusalError builds the refusal a laboratory panics with.
func NewRefusalError(message string) RefusalError {
	return RefusalError{message: message}
}

// Error is the message, and is what both the panic printer and Run report.
func (r RefusalError) Error() string {
	return r.message
}
