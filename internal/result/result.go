package result

type Result[Type any] interface {
	seal() string
	String() string
	IsOk() bool
}

func Ok[Type any](value Type) Result[Type] {
	return ok[Type]{value}
}

func Err[Type any](errorMessage string) Result[Type] {
	return err[Type]{errorMessage}
}

// Output is the value an Ok carries, and empty for an Err.
//
// It exists because a refusal has to be able to show what it refused over. The
// laboratory reads a failing test command as a killed mutant, so when nothing is
// mutated that same failure is a red baseline — and a refusal that does not
// print the command's own output leaves the reader guessing which of a hundred
// reasons a suite might be red. Measured the hard way: an embedded directory
// missing from a sandbox produced a refusal that named neither the file nor the
// pattern, and finding it took four measurements that should have been none.
func Output(res Result[string]) string {
	value, isOk := res.(ok[string])
	if !isOk {
		return ""
	}

	return value.value
}
