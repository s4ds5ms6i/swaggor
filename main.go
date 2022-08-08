package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

const CommentHeader = "SWAGGOR"

type Descriptor struct {
	TaggedInPath    string
	HandlerPath     string
	HandlerFuncName string
	HandlerFuncPos  token.Pos
	HandlerFuncEnd  token.Pos
	tagEnd          token.Pos
	Method          string
	URL             string
	Returns         []string
}

func main() {
	// projectRootPath := "/home/shahram/projects/user-auth/cmd/serve.go"
	// projectRootPath := "/home/shahram/projects/minimal-user-profile/cmd/serve.go"
	projectRootPath := "/home/shahram/projects/minimal-user-profile/"
	projectRootPath = strings.TrimRight(projectRootPath, "/")
	files, err := ioutil.ReadDir(projectRootPath)
	if err != nil {
		log.Fatal(err)
	}

	var projectDirs []string
	for _, f := range files {
		if f.IsDir() && !strings.HasPrefix(f.Name(), ".") {
			projectDirs = append(projectDirs, fmt.Sprintf("%s/%s", projectRootPath, f.Name()))
		}
	}

	packages := make(map[string]*ast.Package)
	for _, dir := range projectDirs {
		fs := token.NewFileSet()
		p, err := parser.ParseDir(fs, dir, nil, 0)
		if err != nil {
			log.Fatal(err)
		}
		for k, v := range p {
			packages[k] = v
		}
	}

	var descriptors []*Descriptor
	for _, pkg := range packages {
		for path, _ := range pkg.Files {
			descList := findTags(path)
			if descList == nil {
				continue
			}

			descriptors = append(descriptors, descList...)
		}
	}

	if descriptors == nil && len(descriptors) == 0 {
		fmt.Printf("No %s tag found.\n", CommentHeader)
		return
	}

	fillInfoFromServeFile(descriptors)
	fillHandlerPath(descriptors, packages)
	fillReturnsOfEachHandler(descriptors)

	fmt.Println(descriptors[0].TaggedInPath, descriptors[0].tagEnd, descriptors[0].URL, descriptors[0].HandlerFuncName,
		descriptors[0].Method, descriptors[0].HandlerPath)

	fmt.Println(descriptors[0].Returns[0])
	fmt.Println("----------------------------------------")

	for _, desc := range descriptors {
		for i := 1; i < len(desc.Returns); i++ {
			errMsg := tryGetErrorMsg(desc.Returns[i])
			if errMsg != "" {
				fmt.Println("500", errMsg)
			} else if strings.Contains(desc.Returns[i], ".JSON(") {
				responseFields := getResponseFields(desc.Returns[0], desc.Returns[i], packages)
				fmt.Println(responseFields)

				for _, field := range responseFields {
					src, err := os.ReadFile(desc.HandlerPath)
					if err != nil {
						log.Fatal(err)
					}

					f, err := getFile(src, 0)
					if err != nil {
						log.Fatal(err)
					}

					ast.Inspect(f, func(n ast.Node) bool {
						switch x := n.(type) {
						case *ast.AssignStmt:
							if x.Pos() >= desc.HandlerFuncPos &&
								x.End() <= desc.HandlerFuncEnd {
								for k, lhs := range x.Lhs {
									start := lhs.Pos() - 1
									end := lhs.End() - 1
									if string(src[start:end]) == field.RawVal {
										start = x.Rhs[k].Pos() - 1
										end = x.Rhs[k].End() - 1
										fmt.Println(string(src[start:end]))
									}
								}
							}
						}
						return true
					})
				}
			} else if strings.Contains(desc.Returns[i], "(") {
				tokens := strings.Split(desc.Returns[i], "(")
				_, b := findDeclPath(packages, tokens[0])
				fmt.Println(getHttpStatusCodeFromReturn(b), tryGetErrorMsg(b))
			}
		}
	}
}

type Field struct {
	Name        string
	Type        string
	IsPrimitive bool
	Attr        string
	JSONName    string
	RawVal      string
	Val         string
}

func getResponseFields(funcBody string, returnStatement string, packages map[string]*ast.Package) []Field {
	rawResponse := strings.TrimRight(getStringAfter(returnStatement, ","), ")")
	if strings.Contains(rawResponse, "{") {
		rawStruct := getStringBefore(rawResponse, "{")
		if strings.Contains(rawStruct, ".") {
			s := getStringAfter(rawStruct, ".")
			_, b := findDeclPath(packages, fmt.Sprintf("type %s struct", s))
			structLines := strings.Split(b, "\n")
			responseLines := strings.Split(strings.TrimRight(getStringAfter(rawResponse, "{"), "}"), ",")
			var processedResponse []Field
			for j, sl := range structLines {
				slTokens := strings.Fields(sl)
				if len(slTokens) != 3 || !strings.Contains(slTokens[2], "json") {
					continue
				}

				var r *Field
				for _, rl := range responseLines {
					rlTokens := strings.Fields(rl)
					if len(rlTokens) > 0 && rlTokens[0] == fmt.Sprintf("%s:", slTokens[0]) {
						r = &Field{
							Name:        slTokens[0],
							Type:        slTokens[1],
							IsPrimitive: isPrimitiveType(slTokens[1]),
							Attr:        slTokens[2],
							JSONName:    getStringInBetween(slTokens[2], "json:\"", "\""),
							RawVal:      rlTokens[1]}
					}
				}

				if r == nil {
					r = &Field{
						Name:        slTokens[0],
						Type:        slTokens[1],
						IsPrimitive: isPrimitiveType(slTokens[1]),
						Attr:        slTokens[2],
						JSONName:    getStringInBetween(slTokens[2], "json:\"", "\""),
						RawVal:      strings.Fields(responseLines[j-1])[0],
					}
				}

				if r.IsPrimitive {
					if strings.Contains(r.RawVal, "http.") {
						if v, ok := httpStatusCodes[strings.TrimLeft(r.RawVal, "http.")]; ok {
							r.Val = strconv.Itoa(v)
						}
					} else {
						r.Val = r.RawVal
					}
				} else {

				}

				processedResponse = append(processedResponse, *r)
			}

			return processedResponse
		}
	}
	return nil
}

func tryGetErrorMsg(src string) string {
	if strings.Contains(src, `fmt.Errorf("`) {
		return getStringInBetween(src, `fmt.Errorf("`, `"`)
	} else if strings.Contains(src, `errors.New("`) {
		return getStringInBetween(src, `errors.New("`, `"`)
	}
	return ""
}

func getHttpStatusCodeFromReturn(returnValue string) string {
	statusCode := getStringInBetween(returnValue, ".JSON(", ",")
	code, ok := httpStatusCodes[strings.TrimLeft(statusCode, "http.")]
	if ok {
		return strconv.Itoa(code)
	} else {
		code, ok = httpStatusCodes[getStringInBetween(returnValue, "http.", ",")]
		if ok {
			return strconv.Itoa(code)
		} else {
			return statusCode
		}
	}
}

func fillReturnsOfEachHandler(descriptors []*Descriptor) {
	for i, desc := range descriptors {
		handlerSrc, err := os.ReadFile(descriptors[i].HandlerPath)
		if err != nil {
			log.Fatal(err)
		}

		f, err := getFile(handlerSrc, 0)
		if err != nil {
			log.Fatal(err)
		}

		var firstFuncDecl *ast.FuncDecl
		var funcFound bool
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if !funcFound {
					if x.Name.Name == descriptors[i].HandlerFuncName {
						funcFound = true
						firstFuncDecl = x
						desc.HandlerFuncPos = x.Pos()
						desc.HandlerFuncEnd = x.End()
					}
				}
			}
			return true
		})

		f, err = getFile(handlerSrc, 0)
		if err != nil {
			log.Fatal(err)
		}

		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ReturnStmt:
				if x.Pos() > firstFuncDecl.Pos() && x.End() < firstFuncDecl.End() {
					start := x.Results[0].Pos() - 1
					end := x.Results[0].End() - 1
					desc.Returns = append(desc.Returns, string(handlerSrc[start:end]))
				}
			}
			return true
		})
	}
}

func fillHandlerPath(descriptors []*Descriptor, packages map[string]*ast.Package) {
	for _, desc := range descriptors {
		desc.HandlerPath, _ = findDeclPath(packages, desc.HandlerFuncName)
	}
}

func findDeclPath(packages map[string]*ast.Package, funcName string) (string, string) {
	for _, pkg := range packages {
		for path, file := range pkg.Files {
			if file.Decls == nil || len(file.Decls) == 0 {
				continue
			}

			src, err := os.ReadFile(path)
			if err != nil {
				log.Fatal(err)
			}

			f, err := getFile(src, 0)
			if err != nil {
				log.Fatal(err)
			}

			var genDeclBody string
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.GenDecl:
					start := x.Pos() - 1
					end := x.End() - 1
					b := string(src[start:end])
					if genDeclBody == "" && strings.Contains(b, funcName) {
						genDeclBody = b
						return true
					}
				}
				return true
			})
			if len(genDeclBody) > 0 {
				return path, genDeclBody
			}

			for _, d := range file.Decls {
				switch x := d.(type) {
				case *ast.FuncDecl:
					if funcName == x.Name.String() {
						return path, ""
					}
				}
			}
		}
	}
	return "", ""
}

var httpStatusCodes = map[string]int{
	"StatusContinue":                      100,
	"StatusSwitchingProtocols":            101,
	"StatusProcessing":                    102,
	"StatusEarlyHints":                    103,
	"StatusOK":                            200,
	"StatusCreated":                       201,
	"StatusAccepted":                      202,
	"StatusNonAuthoritativeInfo":          203,
	"StatusNoContent":                     204,
	"StatusResetContent":                  205,
	"StatusPartialContent":                206,
	"StatusMultiStatus":                   207,
	"StatusAlreadyReported":               208,
	"StatusIMUsed":                        226,
	"StatusMultipleChoices":               300,
	"StatusMovedPermanently":              301,
	"StatusFound":                         302,
	"StatusSeeOther":                      303,
	"StatusNotModified":                   304,
	"StatusUseProxy":                      305,
	"_":                                   306,
	"StatusTemporaryRedirect":             307,
	"StatusPermanentRedirect":             308,
	"StatusBadRequest":                    400,
	"StatusUnauthorized":                  401,
	"StatusPaymentRequired":               402,
	"StatusForbidden":                     403,
	"StatusNotFound":                      404,
	"StatusMethodNotAllowed":              405,
	"StatusNotAcceptable":                 406,
	"StatusProxyAuthRequired":             407,
	"StatusRequestTimeout":                408,
	"StatusConflict":                      409,
	"StatusGone":                          410,
	"StatusLengthRequired":                411,
	"StatusPreconditionFailed":            412,
	"StatusRequestEntityTooLarge":         413,
	"StatusRequestURITooLong":             414,
	"StatusUnsupportedMediaType":          415,
	"StatusRequestedRangeNotSatisfiable":  416,
	"StatusExpectationFailed":             417,
	"StatusTeapot":                        418,
	"StatusMisdirectedRequest":            421,
	"StatusUnprocessableEntity":           422,
	"StatusLocked":                        423,
	"StatusFailedDependency":              424,
	"StatusTooEarly":                      425,
	"StatusUpgradeRequired":               426,
	"StatusPreconditionRequired":          428,
	"StatusTooManyRequests":               429,
	"StatusRequestHeaderFieldsTooLarge":   431,
	"StatusUnavailableForLegalReasons":    451,
	"StatusInternalServerError":           500,
	"StatusNotImplemented":                501,
	"StatusBadGateway":                    502,
	"StatusServiceUnavailable":            503,
	"StatusGatewayTimeout":                504,
	"StatusHTTPVersionNotSupported":       505,
	"StatusVariantAlsoNegotiates":         506,
	"StatusInsufficientStorage":           507,
	"StatusLoopDetected":                  508,
	"StatusNotExtended":                   510,
	"StatusNetworkAuthenticationRequired": 511,
}

func getFile(src []byte, p parser.Mode) (*ast.File, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", src, p)
	return f, err
}

func getStringInBetween(str string, start string, end string) (result string) {
	s := strings.Index(str, start)
	if s == -1 {
		return
	}
	s += len(start)
	e := strings.Index(str[s:], end)
	if e == -1 {
		return
	}
	e += s
	return str[s:e]
}

func getStringBefore(value string, a string) string {
	pos := strings.Index(value, a)
	if pos == -1 {
		return ""
	}
	return value[0:pos]
}

func getStringAfter(value string, a string) string {
	pos := strings.Index(value, a)
	if pos == -1 {
		return ""
	}
	return value[pos+1:]
}

func isPrimitiveType(t string) bool {
	types := map[string]struct{}{
		"complex64":   {},
		"complex128":  {},
		"float32":     {},
		"float64":     {},
		"uint":        {},
		"uint8":       {},
		"uint16":      {},
		"uint32":      {},
		"uint64":      {},
		"int":         {},
		"int8":        {},
		"int16":       {},
		"int32":       {},
		"int64":       {},
		"uintptr":     {},
		"error":       {},
		"bool":        {},
		"string":      {},
		"*complex64":  {},
		"*complex128": {},
		"*float32":    {},
		"*float64":    {},
		"*uint":       {},
		"*uint8":      {},
		"*uint16":     {},
		"*uint32":     {},
		"*uint64":     {},
		"*int":        {},
		"*int8":       {},
		"*int16":      {},
		"*int32":      {},
		"*int64":      {},
		"*uintptr":    {},
		"*error":      {},
		"*bool":       {},
		"*string":     {},
	}
	_, ok := types[t]
	return ok
}

func findAssignment(src []byte,
	lastFuncDecl *ast.FuncDecl,
	lastSelectorExprStr string,
	descriptors []*Descriptor,
	i int) func(n ast.Node) bool {
	return func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if x.Lhs[0].Pos() < lastFuncDecl.Pos() {
				return false
			}

			start := x.Lhs[0].Pos() - 1
			end := x.Lhs[0].End() - 1
			assignmentLeft := string(src[start:end])
			if lastSelectorExprStr == assignmentLeft {
				start = x.Rhs[0].Pos() - 1
				end = x.Rhs[0].End() - 1
				assignmentRight := string(src[start:end])

				descriptors[i].URL = fmt.Sprintf("%s%s",
					getStringInBetween(assignmentRight, "\"", "\""), descriptors[i].URL)

				if strings.Contains(assignmentRight, ".Group(") {
					f, err := getFile(src, 0)
					if err != nil {
						log.Fatal(err)
					}

					ast.Inspect(f, findAssignment(src, lastFuncDecl, getStringBefore(assignmentRight, ".Group("), descriptors, i))
				}
			}
		}

		return true
	}
}

func findTags(path string) (descriptors []*Descriptor) {
	src, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	f, err := getFile(src, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range f.Comments {
		if strings.HasPrefix(c.Text(), CommentHeader) {
			descriptors = append(descriptors,
				&Descriptor{
					tagEnd:       c.End(),
					TaggedInPath: path,
				})
		}
	}

	if descriptors == nil || len(descriptors) == 0 {
		return nil
	} else {
		return descriptors
	}
}

func fillInfoFromServeFile(descriptors []*Descriptor) {
	for i, desc := range descriptors {
		src, err := os.ReadFile(desc.TaggedInPath)
		if err != nil {
			log.Fatal(err)
		}

		f, err := getFile(src, 0)
		if err != nil {
			log.Fatal(err)
		}

		var lastSelectorExpr *ast.SelectorExpr
		var lastCallExpr *ast.CallExpr
		var lastFuncDecl *ast.FuncDecl
		var callerFound, selectorFound bool
		var lastSelectorExprStr string
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if !callerFound {
					lastFuncDecl = x
				}
			case *ast.SelectorExpr:
				if x.Pos() > desc.tagEnd && !callerFound {
					callerFound = true
					lastSelectorExpr = x

					start := lastSelectorExpr.X.Pos() - 1
					end := lastSelectorExpr.X.End() - 1
					lastSelectorExprStr = string(src[start:end])

					start = lastSelectorExpr.Sel.Pos() - 1
					end = lastSelectorExpr.Sel.End() - 1
					descriptors[i].Method = string(src[start:end])
				}
			case *ast.CallExpr:
				if x.Pos() > desc.tagEnd && !selectorFound {
					selectorFound = true
					lastCallExpr = x
				}
			}
			return true
		})

		f, err = getFile(src, 0)
		if err != nil {
			log.Fatal(err)
		}

		for j, arg := range lastCallExpr.Args {
			var argFound bool
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.BasicLit:
					if x.Pos() >= arg.Pos() && !argFound {
						argFound = true
						switch j {
						case 0:
							descriptors[i].URL = getStringInBetween(x.Value, "\"", "\"")
						case 1:
							descriptors[i].HandlerFuncName = getStringInBetween(x.Value, "\"", "\"")
						}
					}
				}
				return true
			})
		}

		f, err = getFile(src, 0)
		if err != nil {
			log.Fatal(err)
		}

		ast.Inspect(f, findAssignment(src, lastFuncDecl, lastSelectorExprStr, descriptors, i))
	}
}
