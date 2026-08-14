package schemata

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Kind is how a mutant gets selected at run time, and there is one per shape of
// expression rather than one per virus.
type Kind int

const (
	// Boolean substitutes an expression of the same type:
	//
	//	a > b   ->   (m == id && a >= b) || (m != id && a > b)
	//
	// Short-circuiting reaches exactly one side, so each operand is evaluated
	// once — measured, because an operand evaluated twice would change any
	// expression whose operands have side effects.
	Boolean Kind = iota

	// Integer substitutes a call, because Go has no conditional expression and
	// an integer literal is not a bool. A call is never a constant, so this does
	// not stand where Go requires one — measured at 3 sites in 91.
	Integer
)

// Gate is one mutant, expressed as an expression that can be swapped for another
// at run time.
type Gate struct {
	Start, End int
	Original   string
	Mutated    string
	Kind       Kind
}

// comparisons yield a bool, which is what lets one expression stand in for two.
var comparisons = map[token.Token]bool{ //nolint:gochecknoglobals // one fixed set, read only
	token.LSS: true, token.LEQ: true, token.GTR: true, token.GEQ: true,
	token.EQL: true, token.NEQ: true,
}

// Expand widens a replacement to the smallest expression that contains it, and
// admits it only if that expression is one a gate can stand in for.
//
// Refusing is not a failure. A refused site keeps the path ditto has always
// taken — its own file, its own compilation — so every mutant that worked before
// still works, which is the constraint this whole change is under. What is
// admitted here is only what has been measured to reach the same verdicts:
// comparisons, and integer literals.
//
// Arithmetic is deliberately absent. `a + b` is an expression, but not a bool,
// and the generated function it would need has not been measured against real
// verdicts. Statements are absent because no expression can replace one.
func Expand(source []byte, replacement Replacement) (Gate, bool) {
	found, base, admitted := locate(source, replacement)
	if !admitted {
		return Gate{}, false
	}

	kind, admitted := kindOf(found)
	if !admitted {
		return Gate{}, false
	}

	start, end := int(found.Pos())-base, int(found.End())-base

	// The mutated expression is the original one with the replacement applied,
	// in the expression's own coordinates.
	mutated := string(source[start:replacement.Start]) + replacement.Mutated +
		string(source[replacement.End:end])

	// A gate splices this text back into the file, so it has to be an expression
	// on its own. Leaving it to the compiler would work — a bad gate fails the
	// shared build — but under one shared compilation that failure takes every
	// other mutant in the run with it, so it is worth refusing early.
	if _, err := parser.ParseExpr(mutated); err != nil {
		return Gate{}, false
	}

	original := string(source[start:end])

	// A difference that is only whitespace is not a mutation. It arises when the
	// original is not gofmt'd and the mutant came back through format.Node, and
	// it was predicted to be refused further up — it was not. Gating it produced
	// a gate whose selected arm behaves exactly like the unselected one: a
	// mutant that can never be killed, and a survivor nobody wrote.
	if fields(original) == fields(mutated) {
		return Gate{}, false
	}

	return Gate{
		Start:    start,
		End:      end,
		Kind:     kind,
		Original: original,
		Mutated:  mutated,
	}, true
}

// locate finds the smallest expression containing the replacement, and refuses
// one that sits where Go requires a constant.
//
// Every gate reads a variable, so no gate is a constant, and in a constant
// context the instrumented file does not compile. Under a single shared build
// that failure takes every other mutant in the run down with it, so refusing
// here is a compilation not wasted.
//
// This sees what the syntax tree shows. A literal that has to adopt another type
// — `float64(ms)/1000`, where the untyped 1000 becomes a float64 — is not
// visible without type information, and is left to the compilation that admits
// the file.
func locate(source []byte, replacement Replacement) (ast.Expr, int, bool) {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, "", source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, 0, false
	}

	base := fileSet.File(file.Pos()).Base()

	var (
		found      ast.Expr
		isConstant bool
	)

	ast.Inspect(file, func(node ast.Node) bool {
		// Inspect calls back with nil on the way out of every subtree.
		if node == nil || !contains(node, base, replacement) {
			return true
		}

		if constantContext(node) {
			isConstant = true
		}

		// Inspect walks outermost first, so every later match is smaller.
		if expression, isExpression := node.(ast.Expr); isExpression {
			found = expression
		}

		return true
	})

	return found, base, found != nil && !isConstant
}

// constantContext reports whether Go requires everything below this node to be
// a constant expression.
func constantContext(node ast.Node) bool {
	switch typed := node.(type) {
	case *ast.GenDecl:
		return typed.Tok == token.CONST
	case *ast.ArrayType:
		return typed.Len != nil
	default:
		return false
	}
}

func contains(node ast.Node, base int, replacement Replacement) bool {
	start, end := int(node.Pos())-base, int(node.End())-base

	return start <= replacement.Start && replacement.End <= end
}

// fields collapses every run of whitespace to one space, so two renderings of
// the same expression compare equal.
func fields(expression string) string {
	return strings.Join(strings.Fields(expression), " ")
}

func kindOf(expression ast.Expr) (Kind, bool) {
	switch typed := expression.(type) {
	case *ast.BinaryExpr:
		return Boolean, comparisons[typed.Op]
	case *ast.BasicLit:
		return Integer, typed.Kind == token.INT
	default:
		return 0, false
	}
}
