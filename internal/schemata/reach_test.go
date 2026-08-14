package schemata_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// The two candidate mechanisms of docs/experiments/refusing-false-kills.md,
// written so that one pass can ask both of them about the same mutant.
//
// Neither is a fix. They answer one question: can a mutant that will not compile
// be recognised without paying the incumbent's subprocess?

// astRefuses is H1 — the syntax tree alone, with no type information and no
// cross-file knowledge.
//
// The four shapes are the ones the population is made of. Each is written to
// prefer a miss over a false positive: refusing a mutant that would have
// compiled removes a real survivor from the score, which is worse than the
// defect being fixed.
func astRefuses(source []byte) bool {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "mutant.go", source, 0)
	if err != nil {
		return true
	}

	refused := false

	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil || refused {
			return !refused
		}

		refused = refusesNode(node)

		return !refused
	})

	return refused || unusedLocal(file)
}

func refusesNode(node ast.Node) bool {
	switch typed := node.(type) {
	case *ast.IndexExpr:
		return negativeLiteral(typed.Index)
	case *ast.SliceExpr:
		return negativeLiteral(typed.Low) || negativeLiteral(typed.High) || negativeLiteral(typed.Max)
	case *ast.BinaryExpr:
		return typed.Op == token.SUB && (holdsString(typed.X) || holdsString(typed.Y))
	case *ast.SwitchStmt:
		return duplicateLiteralCase(typed)
	default:
		return false
	}
}

// negativeLiteral catches `x[-1]`, which is what decrementing a literal zero in
// an index position produces.
func negativeLiteral(expression ast.Expr) bool {
	unary, ok := expression.(*ast.UnaryExpr)
	if !ok || unary.Op != token.SUB {
		return false
	}

	literal, ok := unary.X.(*ast.BasicLit)

	return ok && literal.Kind == token.INT && strings.Trim(literal.Value, "0") != ""
}

// holdsString catches `"a" - x`, which is what turning concatenation into
// subtraction produces. Only a literal proves the type without a type checker,
// so an operand that is merely a string-valued variable is a miss by design.
func holdsString(expression ast.Expr) bool {
	found := false

	ast.Inspect(expression, func(node ast.Node) bool {
		if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			found = true
		}

		return !found
	})

	return found
}

// duplicateLiteralCase catches two case clauses that collide after a literal
// moves. Only literal cases are compared: a named constant would need its value
// resolved, which is not syntax.
func duplicateLiteralCase(statement *ast.SwitchStmt) bool {
	seen := map[string]bool{}

	for _, clause := range statement.Body.List {
		expressions, ok := clause.(*ast.CaseClause)
		if !ok {
			continue
		}

		for _, expression := range expressions.List {
			text, literal := literalText(expression)
			if !literal {
				continue
			}

			if seen[text] {
				return true
			}

			seen[text] = true
		}
	}

	return false
}

func literalText(expression ast.Expr) (string, bool) {
	switch typed := expression.(type) {
	case *ast.BasicLit:
		return typed.Value, true
	case *ast.UnaryExpr:
		if inner, ok := typed.X.(*ast.BasicLit); ok {
			return typed.Op.String() + inner.Value, true
		}
	}

	return "", false
}

// unusedLocal catches a variable left unread after a comparison became a
// constant. It is scope analysis over the tree with no types: a declared name is
// looked for by name inside the body that declares it, and any occurrence at all
// counts as a use, so a shadowed or reassigned name is a miss rather than a
// false positive.
func unusedLocal(file *ast.File) bool {
	unused := false

	ast.Inspect(file, func(node ast.Node) bool {
		body := functionBody(node)
		if body == nil || unused {
			return !unused
		}

		for _, name := range declaredIn(body) {
			if occurrences(body, name) <= declarations(body, name) {
				unused = true

				return false
			}
		}

		return true
	})

	return unused
}

func functionBody(node ast.Node) *ast.BlockStmt {
	switch typed := node.(type) {
	case *ast.FuncDecl:
		return typed.Body
	case *ast.FuncLit:
		return typed.Body
	default:
		return nil
	}
}

// declaredIn lists the names a body introduces with `:=` or `var`. Go only
// refuses an unused *local*, so nothing outside a body is collected.
func declaredIn(body *ast.BlockStmt) []string {
	names := []string{}

	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			if typed.Tok == token.DEFINE {
				names = append(names, identNames(typed.Lhs)...)
			}
		case *ast.ValueSpec:
			for _, name := range typed.Names {
				if name.Name != "_" {
					names = append(names, name.Name)
				}
			}
		}

		return true
	})

	return names
}

func identNames(expressions []ast.Expr) []string {
	names := []string{}

	for _, expression := range expressions {
		if ident, ok := expression.(*ast.Ident); ok && ident.Name != "_" {
			names = append(names, ident.Name)
		}
	}

	return names
}

func occurrences(body *ast.BlockStmt, name string) int {
	count := 0

	ast.Inspect(body, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok && ident.Name == name {
			count++
		}

		return true
	})

	return count
}

// declarations counts the appearances of a name on the left of `:=` or in a
// `var`, so that a name appearing only there is the one Go refuses.
func declarations(body *ast.BlockStmt, name string) int {
	count := 0

	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			if typed.Tok == token.DEFINE {
				count += countNamed(typed.Lhs, name)
			}
		case *ast.ValueSpec:
			for _, declared := range typed.Names {
				if declared.Name == name {
					count++
				}
			}
		}

		return true
	})

	return count
}

func countNamed(expressions []ast.Expr, name string) int {
	count := 0

	for _, expression := range expressions {
		if ident, ok := expression.(*ast.Ident); ok && ident.Name == name {
			count++
		}
	}

	return count
}
