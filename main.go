package main

import (
	"fmt"

	"io/ioutil"
	"os"

	"github.com/k0kubun/pp"
)

func main() {
	var src string

	if len(os.Args) > 1 {
		filename := os.Args[1]
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Println(err)
		}

		src = string(data)
	} else {
		data, _ := ioutil.ReadAll(os.Stdin)
		src = string(data)
	}

	statements, err := Parse(src)
	if err != nil {
		fmt.Println(err)
		return
	}

	for i, statement := range statements {
		statements[i] = Walk(statement)
	}

	prelude, _ := Parse("void print(int i);\n")
	statements = append(prelude, statements...)

	env := &Env{}
	errs := Analyze(statements, env)
	if len(errs) > 0 {
		Exit(src, errs)
	}

	err = CheckType(statements)
	if err != nil {
		Exit(src, []error{err})
	}

	pp.Println(CompileIR(statements))
	fmt.Println(CompileIR(statements))
}

func Exit(src string, errs []error) {
	for _, err := range errs {
		switch e := err.(type) {
		case SemanticError:
			lineNumber, columnNumber := posToLineInfo(src, int(e.Pos))
			err = fmt.Errorf("%d:%d: %v", lineNumber, columnNumber, e.Err)

		default:
		}

		fmt.Fprintln(os.Stderr, err)
	}

	os.Exit(1)
}
