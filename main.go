package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/structtag"
)

const (
	tagNameRequire     = "require"
	tagValueAssignment = "assignment"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: ")
		fmt.Println("gocontract file.go")
		os.Exit(1)
	}

	_, debugLog := os.LookupEnv("DEBUG")

	filename := os.Args[1]

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	structs, err := parseStructs(file)
	if err != nil {
		log.Fatal(err)

	}
	if debugLog {
		fmt.Printf("structs: %v\n", structs)
	}

	methodNames := map[string]struct{}{}
	for _, struc := range structs {
		for _, methods := range struc.AssignmentRequired {
			for name := range methods {
				methodNames[name] = struct{}{}
			}
		}
	}
	if debugLog {
		fmt.Printf("methodNames: %v\n", methodNames)
	}

	bodies := parseMethods(file, methodNames)
	if debugLog {
		fmt.Printf("bodies: %v\n", bodies)
	}
	hasError := false
	for _, struc := range structs {
		for field, methods := range struc.AssignmentRequired {
			for name := range methods {
				if bodies[name] == nil {
					fmt.Printf("%v uninitialized struct field %v.%v in %v, method not found.\n", filename, struc.Name, field, name)
					hasError = true
					continue
				}
				if !isInitialized(bodies[name], struc.Name, field) {
					fmt.Printf("%v uninitialized struct field %v.%v in %v\n", filename, struc.Name, field, name)
					hasError = true
					continue
				}
			}
		}
	}

	if hasError {
		os.Exit(1)
	}
}

type structInfo struct {
	Name               string
	AssignmentRequired map[string]map[string]struct{}
}

func isInitialized(body *ast.BlockStmt, struc string, field string) (isInited bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		if isInited {
			return false
		}

		compLit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		ident, ok := compLit.Type.(*ast.Ident)
		if !ok {
			return true
		}

		if ident.Name != struc {
			return true
		}

		for _, elem := range compLit.Elts {
			if kve, ok := elem.(*ast.KeyValueExpr); ok {
				if ident, ok := kve.Key.(*ast.Ident); ok {
					if ident.Name == field {
						isInited = true
						return false
					}
				}
			}
		}

		return true
	})

	return isInited
}

func parseMethods(file *ast.File, names map[string]struct{}) (bodies map[string]*ast.BlockStmt) {
	bodies = map[string]*ast.BlockStmt{}
	ast.Inspect(file, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		if _, ok := names[funcDecl.Name.Name]; !ok {
			return true
		}

		bodies[funcDecl.Name.Name] = funcDecl.Body

		return true
	})

	return bodies
}

func parseStructs(file *ast.File) (structs []structInfo, err error) {
	structs = []structInfo{}
	ast.Inspect(file, func(n ast.Node) bool {
		if err != nil {
			return false
		}

		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		struc, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return false
		}

		for _, item := range struc.Fields.List {
			if item.Tag == nil {
				continue
			}

			unqouted, err := strconv.Unquote(item.Tag.Value)
			if err != nil {
				continue
			}

			tags, err := structtag.Parse(unqouted)
			if err != nil {
				continue
			}

			tag, err := tags.Get(tagNameRequire)
			if err != nil {
				continue
			}

			if tag == nil {
				continue
			}

			if tag.Name != tagValueAssignment {
				continue
			}

			info := structInfo{
				Name:               typeSpec.Name.Name,
				AssignmentRequired: map[string]map[string]struct{}{},
			}
			for _, option := range tag.Options {
				var name string
				if len(item.Names) > 0 {
					name = item.Names[0].Name
				} else {
					// Inherited type.
					switch n := item.Type.(type) {
					case *ast.Ident:
						name = n.Name
					case *ast.StarExpr:
						switch x := n.X.(type) {
						case *ast.Ident:
							name = x.Name
						case *ast.SelectorExpr:
							name = x.Sel.Name
						}
					}
				}

				methods := info.AssignmentRequired[name]
				if methods == nil {
					methods = map[string]struct{}{}
				}

				methodName := strings.TrimSpace(option)
				methods[methodName] = struct{}{}

				info.AssignmentRequired[name] = methods
			}
			structs = append(structs, info)
		}

		return true
	})

	return structs, err
}
