package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// intermediate representation
type IRProgram struct {
	Declarations []*IRVariableDeclaration
	Functions    []*IRFunctionDefinition
}

type traverseAction (func(statement IRStatement) IRStatement)

func Traverse(statement IRStatement, action traverseAction) IRStatement {
	switch s := statement.(type) {
	case *IRFunctionDefinition:
		body := Traverse(s.Body, action)
		if body != nil {
			s.Body = body
			return s
		}

		return nil

	case *IRCompoundStatement:
		statements := []IRStatement{}
		for _, statement := range s.Statements {
			transformed := Traverse(statement, action)
			if transformed != nil {
				statements = append(statements, transformed)
			}
		}

		s.Statements = statements

		return action(s)

	default:
		return action(statement)
	}
}

func (s *IRProgram) String() string {
	var declStrs []string
	for _, decl := range s.Declarations {
		declStrs = append(declStrs, decl.String())
	}

	var stmtStrs []string
	for _, statement := range s.Functions {
		stmtStrs = append(stmtStrs, statement.String())
	}

	return strings.Join(declStrs, "\n") + "\n\n" + strings.Join(stmtStrs, "\n\n")
}

type IRStatement interface {
	String() string
}
type IRExpression interface {
	String() string
}

type IRVariableDeclaration struct {
	Var *Symbol
}

func (s *IRVariableDeclaration) String() string {
	return fmt.Sprintf("%v %v", s.Var.Type, s.Var.Name)
}

type IRFunctionDefinition struct {
	Var        *Symbol
	Parameters []*IRVariableDeclaration
	Body       IRStatement
	VarSize    int
}

func (s *IRFunctionDefinition) String() string {
	var params []string
	for _, p := range s.Parameters {
		params = append(params, p.String())
	}

	return fmt.Sprintf("%v(%v)\n%v", s.Var.Name, strings.Join(params, ", "), s.Body)
}

type IRAssignmentStatement struct {
	Var        *Symbol
	Expression IRExpression
}

func (s *IRAssignmentStatement) String() string {
	return fmt.Sprintf("%v = %v", s.Var.Name, s.Expression)
}

type IRWriteStatement struct {
	Dest *Symbol
	Src  *Symbol
}

func (s *IRWriteStatement) String() string {
	return fmt.Sprintf("*%v = %v", s.Dest.Name, s.Src.Name)
}

type IRReadStatement struct {
	Dest *Symbol
	Src  *Symbol
}

func (s *IRReadStatement) String() string {
	return fmt.Sprintf("%v = *%v", s.Dest.Name, s.Src.Name)
}

type IRLabelStatement struct {
	Name string
}

func (s *IRLabelStatement) String() string {
	return fmt.Sprintf("%s:", s.Name)
}

type IRIfStatement struct {
	Var        *Symbol
	TrueLabel  string
	FalseLabel string
}

func (s *IRIfStatement) String() string {
	if len(s.FalseLabel) == 0 {
		return fmt.Sprintf("if (%s) %s", s.Var.Name, s.TrueLabel)
	}

	return fmt.Sprintf("if (%s) %s else %s", s.Var.Name, s.TrueLabel, s.FalseLabel)
}

type IRGotoStatement struct {
	Label string
}

func (s *IRGotoStatement) String() string {
	return fmt.Sprintf("goto %s", s.Label)
}

type IRCallStatement struct {
	Dest *Symbol
	Func *Symbol
	Vars []*Symbol
}

func (s *IRCallStatement) String() string {
	var args []string
	for _, symbol := range s.Vars {
		args = append(args, symbol.Name)
	}

	return fmt.Sprintf("%s = %s(%s)", s.Dest.Name, s.Func.Name, strings.Join(args, ", "))
}

type IRReturnStatement struct {
	Var *Symbol
}

func (s *IRReturnStatement) String() string {
	return fmt.Sprintf("return %s", s.Var.Name)
}

type IRSystemCallStatement struct {
	Name string
	Var  *Symbol
}

func (s *IRSystemCallStatement) String() string {
	return fmt.Sprintf("%v(%s)", s.Name, s.Var.Name)
}

type IRCompoundStatement struct {
	Declarations []*IRVariableDeclaration
	Statements   []IRStatement
}

func (s *IRCompoundStatement) String() string {
	var declStrs []string
	for _, decl := range s.Declarations {
		declStrs = append(declStrs, decl.String())
	}

	var stmtStrs []string
	for _, statement := range s.Statements {
		stmtStrs = append(stmtStrs, statement.String())
	}

	str := ""
	if len(declStrs) > 0 {
		str += strings.Join(declStrs, "\n") + "\n"
	}
	str += strings.Join(stmtStrs, "\n")

	return str
}

// IRExpression

type IRVariableExpression struct {
	Var *Symbol
}

func (e *IRVariableExpression) String() string {
	return e.Var.Name
}

type IRNumberExpression struct {
	Value int
}

func (e *IRNumberExpression) String() string {
	return strconv.Itoa(e.Value)
}

type IRBinaryExpression struct {
	Operator string
	Left     IRExpression
	Right    IRExpression
}

func (e *IRBinaryExpression) String() string {
	return fmt.Sprintf("(%s %v %v)", e.Operator, e.Left, e.Right)
}

type IRAddressExpression struct {
	Var *Symbol
}

func (e *IRAddressExpression) String() string {
	return fmt.Sprintf("&%v", e.Var.Name)
}

var counter = map[string]int{}

func label(name string) string {
	labelName := fmt.Sprintf("%s_%d", name, counter[name])
	counter[name]++

	return labelName
}

func tmpvar() *Symbol {
	return &Symbol{
		Name:  label("#tmp"),
		Level: 2, // not global
		Type:  Int(),
	}
}

// CompileIR convert Statements to intermediate representation
func CompileIR(statements []Statement) *IRProgram {
	var decls []*IRVariableDeclaration
	var funcs []*IRFunctionDefinition

	var irStatements []IRStatement
	for _, statement := range statements {
		switch s := statement.(type) {
		case *Declaration:
			symbols := findSymbolsFromDeclaration(s)
			decls = append(decls, IRVariableDeclarations(symbols)...)
		default:
			irStatements = append(irStatements, compileIRStatement(s))
		}
	}

	for _, statement := range irStatements {
		switch s := statement.(type) {
		case *IRFunctionDefinition:
			funcs = append(funcs, s)
		case *IRVariableDeclaration:
			decls = append(decls, s)
		}
	}

	return &IRProgram{
		Declarations: decls,
		Functions:    funcs,
	}
}

func compileIRStatement(statement Statement) IRStatement {
	if statement == nil {
		return nil
	}

	switch s := statement.(type) {
	case *FunctionDefinition:
		if s.Statement == nil {
			return nil
		}

		identifier := findIdentifierExpression(s.Identifier)

		var paramSymbols []*Symbol
		for _, p := range s.Parameters {
			parameter, ok := p.(*ParameterDeclaration)
			if ok {
				identifier := findIdentifierExpression(parameter.Identifier)
				paramSymbols = append(paramSymbols, identifier.Symbol)
			}
		}

		return &IRFunctionDefinition{
			Var:        identifier.Symbol,
			Parameters: IRVariableDeclarations(paramSymbols),
			Body:       compileIRStatement(s.Statement),
		}

	case *CompoundStatement:
		var symbols []*Symbol
		for _, d := range s.Declarations {
			declaration, ok := d.(*Declaration)
			if ok {
				symbols = append(symbols, findSymbolsFromDeclaration(declaration)...)
			}
		}

		var statements []IRStatement
		for _, statement := range s.Statements {
			statements = append(statements, compileIRStatement(statement))
		}

		return &IRCompoundStatement{
			Declarations: IRVariableDeclarations(symbols),
			Statements:   statements,
		}

	case *ExpressionStatement:
		if s.Value == nil {
			return nil
		}

		switch e := s.Value.(type) {
		case *ExpressionList:
			var statements []IRStatement
			for _, value := range e.Values {
				statements = append(statements, compileIRStatement(&ExpressionStatement{Value: value}))
			}

			return &IRCompoundStatement{
				Statements: statements,
			}

		case *FunctionCallExpression:
			name := findIdentifierExpression(e.Identifier).Name

			if isSystemCall(name) {
				tmp := tmpvar()
				arg, decls, beforeArg := compileIRExpression(e.Argument)

				return &IRCompoundStatement{
					Declarations: append(decls, &IRVariableDeclaration{Var: tmp}),
					Statements: append(beforeArg,
						&IRAssignmentStatement{
							Var:        tmp,
							Expression: arg,
						},
						&IRSystemCallStatement{
							Name: name,
							Var:  tmp,
						},
					),
				}
			}
		}

		_, decls, beforeValue := compileIRExpression(s.Value)

		return &IRCompoundStatement{
			Declarations: decls,
			Statements:   beforeValue,
		}

	case *IfStatement:
		conditionVar := tmpvar()

		trueLabel := label("true")
		falseLabel := label("false")
		endLabel := label("end")

		condition, decls, beforeCondition := compileIRExpression(s.Condition)

		statements := []IRStatement{
			&IRAssignmentStatement{
				Var:        conditionVar,
				Expression: condition,
			},
			&IRIfStatement{
				Var:        conditionVar,
				TrueLabel:  trueLabel,
				FalseLabel: falseLabel,
			},
			&IRLabelStatement{Name: trueLabel},
			compileIRStatement(s.TrueStatement),
			&IRGotoStatement{Label: endLabel},
			&IRLabelStatement{Name: falseLabel},
		}

		if s.FalseStatement != nil {
			statements = append(statements, compileIRStatement(s.FalseStatement))
		}

		statements = append(statements, &IRLabelStatement{Name: endLabel})

		return &IRCompoundStatement{
			Declarations: append(IRVariableDeclarations([]*Symbol{conditionVar}), decls...),
			Statements:   append(beforeCondition, statements...),
		}

	case *WhileStatement:
		conditionVar := tmpvar()

		beginLabel := label("while_begin")
		endLabel := label("while_end")

		condition, decls, beforeCondition := compileIRExpression(s.Condition)
		statements := append([]IRStatement{&IRLabelStatement{Name: beginLabel}}, beforeCondition...)

		statements = append(statements,
			&IRAssignmentStatement{
				Var:        conditionVar,
				Expression: condition,
			},
			&IRIfStatement{
				Var:        conditionVar,
				FalseLabel: endLabel,
			},
			compileIRStatement(s.Statement),
			&IRGotoStatement{Label: beginLabel},
			&IRLabelStatement{Name: endLabel},
		)

		return &IRCompoundStatement{
			Declarations: append(IRVariableDeclarations([]*Symbol{conditionVar}), decls...),
			Statements:   statements,
		}

	case *ReturnStatement:
		// return exp;
		//
		// tmp = <exp>
		// return tmp

		if s.Value == nil {
			return &IRReturnStatement{}
		}

		tmp := tmpvar()

		value, decls, beforeValue := compileIRExpression(s.Value)
		return &IRCompoundStatement{
			Declarations: append(IRVariableDeclarations([]*Symbol{tmp}), decls...),
			Statements: append(beforeValue,
				&IRAssignmentStatement{Var: tmp, Expression: value},
				&IRReturnStatement{Var: tmp},
			),
		}

	default:
		panic("unexpected statement")
	}
}

func IRVariableDeclarations(symbols []*Symbol) []*IRVariableDeclaration {
	var declarations []*IRVariableDeclaration
	for _, symbol := range symbols {
		declarations = append(declarations, &IRVariableDeclaration{
			Var: symbol,
		})
	}

	return declarations
}

func findSymbolsFromDeclaration(declaration *Declaration) []*Symbol {
	var symbols []*Symbol
	for _, declarator := range declaration.Declarators {
		identifier := findIdentifierExpression(declarator.Identifier)
		symbols = append(symbols, identifier.Symbol)
	}

	return symbols
}

func compileIRExpression(expression Expression) (IRExpression, []*IRVariableDeclaration, []IRStatement) {
	switch e := expression.(type) {
	case *NumberExpression:
		value, _ := strconv.Atoi(e.Value)
		return &IRNumberExpression{
			Value: value,
		}, nil, nil

	case *IdentifierExpression:
		return &IRVariableExpression{
			Var: e.Symbol,
		}, nil, nil

	case *UnaryExpression:
		if e.Operator == "*" {
			result := tmpvar()
			tmp := tmpvar()
			irValue, decls, beforeValue := compileIRExpression(e.Value)

			statements := []IRStatement{
				&IRAssignmentStatement{
					Var:        tmp,
					Expression: irValue,
				},
				&IRReadStatement{Dest: result, Src: tmp},
			}

			decls = append(IRVariableDeclarations([]*Symbol{result, tmp}), decls...)
			statements = append(beforeValue, statements...)

			return &IRVariableExpression{
				Var: result,
			}, decls, statements
		}

		if e.Operator == "&" {
			value, decls, statements := compileIRExpression(e.Value)
			v, _ := value.(*IRVariableExpression)

			return &IRAddressExpression{
				Var: v.Var,
			}, decls, statements
		}

	case *BinaryExpression:
		// return (a || b) && c
		// v;
		// if (a) {
		//   v = 1
		// } else if (b) {
		//   v = 1;
		// } else {
		//   v = 0;
		// }
		// int v;
		// if (a) {
		//   if (b) {
		//     v = 1;
		//   } else {
		//     v = 0;
		//   }
		// } else {
		//   v = 0;
		// }

		if e.IsAssignment() {
			// a = (b = c);
			// *(p + 2) = 4
			switch left := e.Left.(type) {
			case *UnaryExpression:
				// address = left
				// tmpRight = exp
				// *address = tmpRight
				if left.Operator == "*" {
					address := tmpvar()
					tmpRight := tmpvar()

					right, rightDecls, beforeRight := compileIRExpression(e.Right)
					leftExpression, leftDecls, beforeLeft := compileIRExpression(left.Value)

					decls := append(rightDecls, leftDecls...)
					decls = append(IRVariableDeclarations([]*Symbol{address, tmpRight}), decls...)

					statements := []IRStatement{
						&IRAssignmentStatement{
							Var:        address,
							Expression: leftExpression,
						},
						&IRAssignmentStatement{
							Var:        tmpRight,
							Expression: right,
						},
						&IRWriteStatement{Dest: address, Src: tmpRight},
					}
					statements = append(append(beforeLeft, beforeRight...), statements...)

					return &IRVariableExpression{tmpRight}, decls, statements
				}

			default:
				tmp := tmpvar()
				decls := IRVariableDeclarations([]*Symbol{tmp})

				symbol := findIdentifierExpression(e.Left).Symbol
				right, rightDecls, beforeRight := compileIRExpression(e.Right)

				decls = append(decls, rightDecls...)
				statements := append(beforeRight,
					&IRAssignmentStatement{
						Var:        symbol,
						Expression: right,
					},
				)

				return &IRVariableExpression{Var: symbol}, decls, statements
			}
		}

		switch e.Operator {
		case "&&":
			tmp := tmpvar()

			decls := IRVariableDeclarations([]*Symbol{tmp})
			statements := []IRStatement{
				compileIRStatement(&IfStatement{
					Condition: e.Left,
					TrueStatement: &IfStatement{
						Condition:      e.Right,
						TrueStatement:  assignStatementBySymbol(tmp, 1),
						FalseStatement: assignStatementBySymbol(tmp, 0),
					},
					FalseStatement: assignStatementBySymbol(tmp, 0),
				}),
			}

			return &IRVariableExpression{Var: tmp}, decls, statements

		case "||":
			tmp := tmpvar()

			decls := IRVariableDeclarations([]*Symbol{tmp})
			statements := []IRStatement{
				compileIRStatement(&IfStatement{
					Condition:     e.Left,
					TrueStatement: assignStatementBySymbol(tmp, 1),
					FalseStatement: &IfStatement{
						Condition:      e.Right,
						TrueStatement:  assignStatementBySymbol(tmp, 1),
						FalseStatement: assignStatementBySymbol(tmp, 0),
					},
				}),
			}

			return &IRVariableExpression{Var: tmp}, decls, statements
		}

		left, leftDecls, beforeLeft := compileIRExpression(e.Left)
		right, rightDecls, beforeRight := compileIRExpression(e.Right)

		t, _ := typeOfExpression(e)
		switch t.(type) {
		case PointerType:
			leftType, _ := typeOfExpression(e.Left)

			if _, isInt := leftType.(BasicType); isInt {
				// 4 * r + l
				left = &IRBinaryExpression{
					Operator: "*",
					Left:     &IRNumberExpression{Value: 4}, // int -> 4 bytes
					Right:    left,
				}
			} else {
				// l + 4 * r
				right = &IRBinaryExpression{
					Operator: "*",
					Left:     &IRNumberExpression{Value: 4}, // int -> 4 bytes
					Right:    right,
				}
			}
		}

		return &IRBinaryExpression{
			Operator: e.Operator,
			Left:     left,
			Right:    right,
		}, append(leftDecls, rightDecls...), append(beforeLeft, beforeRight...)

	case *FunctionCallExpression:
		funcIdentifier := findIdentifierExpression(e.Identifier)

		var args []Expression
		switch arg := e.Argument.(type) {
		case *ExpressionList:
			args = arg.Values
		default:
			if arg != nil {
				args = []Expression{arg}
			}
		}

		var argSymbols []*Symbol
		var statements []IRStatement
		var decls []*IRVariableDeclaration

		for _, arg := range args {
			tmp := tmpvar()
			argSymbols = append(argSymbols, tmp)

			expression, expressionDecls, beforeExpression := compileIRExpression(arg)

			decls = append(decls, expressionDecls...)

			statements = append(statements, beforeExpression...)
			statements = append(statements, &IRAssignmentStatement{
				Var:        tmp,
				Expression: expression,
			})
		}

		result := tmpvar()

		// result = f(a0, a1, ...)
		statements = append(statements, &IRCallStatement{
			Dest: result,
			Func: funcIdentifier.Symbol,
			Vars: argSymbols,
		})

		decls = append(decls, IRVariableDeclarations(append(argSymbols, result))...)
		return &IRVariableExpression{
			Var: result,
		}, decls, statements

	}

	panic(fmt.Sprintf("unexpected expression: `%v`", reflect.TypeOf(expression)))
}

func assignStatementBySymbol(symbol *Symbol, value int) *ExpressionStatement {
	return &ExpressionStatement{
		Value: &BinaryExpression{
			Operator: "=",
			Left:     &IdentifierExpression{Symbol: symbol},
			Right:    &NumberExpression{Value: strconv.Itoa(value)},
		},
	}
}

func isSystemCall(name string) bool {
	switch name {
	case "print", "putchar":
		return true
	}

	return false
}
