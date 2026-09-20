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

// Output is what the command printed, whichever way it ended.
//
// It answered only for an Ok until the command's own package scope had to be
// read: that scope is announced by `go test -json` on the stream of a GREEN
// suite, and a green suite is an Err — so the one run whose output the scope
// check needs was the one this refused to answer for. The name and the contract
// now say the same thing: the output belongs to the command, not to the verdict.
//
// What the Ok side is for is unchanged, and it was measured the hard way: the
// laboratory reads a failing command as a killed mutant, so a refusal to score a
// red baseline has to show the command's own words — the first version named
// neither the file nor the pattern, and finding the embedded directory behind it
// took four measurements that should have been none.
func Output(res Result[string]) string {
	if ended, isOk := res.(ok[string]); isOk {
		return ended.value
	}

	if ended, isErr := res.(err[string]); isErr {
		return ended.errorMessage
	}

	// Unreachable: the seal keeps every implementation inside this package.
	return ""
}
