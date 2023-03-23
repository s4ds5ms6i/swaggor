package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"gitlab.snapp.ir/security_regulatory/swaggor/util"
)

const CommentHeader = "SWAGGOR"

type Descriptor struct {
	TaggedInPath     string
	TagEnd           token.Pos
	Handler          Handler
	middlewaresStart token.Pos
	Middlewares      []Middleware
	Headers          []Header
}

type Handler struct {
	HandlerPath     string
	HandlerFuncName string
	HandlerFuncPos  token.Pos
	HandlerFuncEnd  token.Pos
	Method          string
	URL             string
	RawReturns      []string
	Returns         []Return
}

type Return struct {
	StatusCode string
	JSON       string
	Message    string
}

type Middleware struct {
	Name string
	Path string
}

type Header struct {
	Decl  string
	Value string
}

func main() {
	projectRootPath := "/home/shahram/projects/minimal-user-profile"
	excludedPaths := map[string]struct{}{
		"vendor": {},
		"assets": {},
		"docs":   {},
	}
	projectRootPath = strings.TrimRight(projectRootPath, "/")
	files, err := ioutil.ReadDir(projectRootPath)
	if err != nil {
		log.Fatal(err)
	}

	var projectDirs []string
	for _, f := range files {
		if _, ok := excludedPaths[f.Name()]; f.IsDir() && !strings.HasPrefix(f.Name(), ".") && !ok {
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

	fillHandler(descriptors)
	fillMiddlewares(descriptors)
	fillHandlerPath(descriptors, packages)
	fillMiddlewaresPath(descriptors, packages)
	fillReturnsOfEachHandler(descriptors)
	fillHeadersOfMiddleware(descriptors, packages)

	// fmt.Println(descriptors[0].TaggedInPath, descriptors[0].tagEnd, descriptors[0].URL, descriptors[0].Handler.HandlerFuncName,
	// 	descriptors[0].Handler.Method, descriptors[0].Handler.HandlerPath)
	//
	// fmt.Println(descriptors[0].Handler.RawReturns[0])
	// fmt.Println("----------------------------------------")

	descIndex := 0
	for _, desc := range descriptors {
		for i := 1; i < len(desc.Handler.RawReturns); i++ {
			errMsg := tryGetErrorMsg(desc.Handler.RawReturns[i])
			if errMsg != "" {
				descriptors[descIndex].Handler.Returns = append(descriptors[descIndex].Handler.Returns, Return{StatusCode: "500", Message: errMsg})
			} else if strings.Contains(desc.Handler.RawReturns[i], ".JSON(") {
				responseFields := getFieldsFromReturn(desc, desc.Handler.RawReturns[i], packages)
				json := "{\n"
				for _, field := range responseFields {
					if json != "{\n" && !strings.HasSuffix(json, ",\n") {
						json = fmt.Sprintf("%s,\n", strings.TrimRight(json, "\n"))
					}

					json = json + getJSONBody(packages, field, field.IsArray)
				}

				json = fmt.Sprintf("%s\n}", strings.TrimRight(json, ",\n"))

				descriptors[descIndex].Handler.Returns = append(descriptors[descIndex].Handler.Returns, Return{StatusCode: "200", JSON: json})
			} else if strings.Contains(desc.Handler.RawReturns[i], "(") {
				tokens := strings.Split(desc.Handler.RawReturns[i], "(")
				_, b := findDeclPath(packages, tokens[0])
				descriptors[descIndex].Handler.Returns = append(descriptors[descIndex].Handler.Returns,
					Return{
						StatusCode: getHttpStatusCodeFromReturn(b),
						Message:    tryGetErrorMsg(b),
					})
			}
		}

		sort.Slice(descriptors[descIndex].Handler.Returns[:], func(i, j int) bool {
			return descriptors[descIndex].Handler.Returns[i].StatusCode < descriptors[descIndex].Handler.Returns[j].StatusCode
		})
		// fmt.Println(descriptors[descIndex].Handler.Returns)
		// fmt.Println("============================")
		descIndex++
	}

	var swagger string
	swagger = fmt.Sprintln(
		`openapi: 3.0.0
info:
  title: Title of the service
  description: |
    Service description.
  version: '0.0.0'
paths:`)
	for _, desc := range descriptors {
		var distinctReturns []Return
		for _, ret := range desc.Handler.Returns {
			found := false
			for i, distRet := range distinctReturns {
				if distRet.StatusCode == ret.StatusCode {
					found = true
					if !util.IsEmptyOrWhitespace(ret.Message) {
						distinctReturns[i].Message = fmt.Sprintf("%s / %s", distinctReturns[i].Message, ret.Message)
					}
					break
				}
			}

			if found == false {
				distinctReturns = append(distinctReturns, ret)
			}
		}

		swagger = fmt.Sprintf("%s%s'%s':\n", swagger, indent(2), desc.Handler.URL)
		swagger = fmt.Sprintf("%s%s%s:\n", swagger, indent(4), strings.ToLower(desc.Handler.Method))
		swagger = fmt.Sprintf("%s%ssummary: %s\n", swagger, indent(6), "Some description")
		swagger = fmt.Sprintf("%s%sdescription: %s\n", swagger, indent(6), "Some description")
		queryParams := getQueryParams(desc, packages)
		if len(queryParams) > 0 {
			swagger = fmt.Sprintf("%s%sparameters:\n", swagger, indent(6))
			for p, t := range queryParams {
				swagger = fmt.Sprintf("%s%s- in: query\n", swagger, indent(8))
				swagger = fmt.Sprintf("%s%sname: %s\n", swagger, indent(10), p)
				swagger = fmt.Sprintf("%s%sschema: \n", swagger, indent(10))
				swagger = fmt.Sprintf("%s%stype: %s\n", swagger, indent(12), t)
			}
		}

		reqHeaders := getRequestHeaders(desc, packages)
		if len(reqHeaders) > 0 {
			if len(queryParams) == 0 {
				swagger = fmt.Sprintf("%s%sparameters:\n", swagger, indent(6))
			}

			for h, t := range reqHeaders {
				swagger = fmt.Sprintf("%s%s- in: header\n", swagger, indent(8))
				swagger = fmt.Sprintf("%s%sname: %s\n", swagger, indent(10), h)
				swagger = fmt.Sprintf("%s%sschema: \n", swagger, indent(10))
				swagger = fmt.Sprintf("%s%stype: %s\n", swagger, indent(12), t)
			}
		}

		reqInputsFromContext := getFromContext(desc, packages)
		if len(reqInputsFromContext) > 0 {
			if len(queryParams) == 0 && len(reqHeaders) == 0 {
				swagger = fmt.Sprintf("%s%sparameters:\n", swagger, indent(6))
			}
		}

		if len(desc.Headers) > 0 {
			for _, h := range desc.Headers {
				swagger = fmt.Sprintf("%s%s- in: header\n", swagger, indent(8))
				swagger = fmt.Sprintf("%s%sname: %s\n", swagger, indent(10), h.Value)
				swagger = fmt.Sprintf("%s%sdescription: API expects %s to be included in request headers.\n", swagger, indent(10), h.Value)
				swagger = fmt.Sprintf("%s%sschema: \n", swagger, indent(10))
				swagger = fmt.Sprintf("%s%stype: %s\n", swagger, indent(12), "string") // TODO: Are all the header types string?
				swagger = fmt.Sprintf("%s%srequired: true\n", swagger, indent(10))     // TODO: Are all the headers mandatory?
			}
		}

		swagger = fmt.Sprintf("%s%sresponses:\n", swagger, indent(6))
		for _, ret := range distinctReturns {
			swagger = fmt.Sprintf("%s%s'%s':\n", swagger, indent(8), ret.StatusCode)
			if len(ret.Message) > 0 {
				swagger = fmt.Sprintf("%s%sdescription: %s\n", swagger, indent(10), ret.Message)
			} else {
				swagger = fmt.Sprintf("%s%sdescription: %s\n", swagger, indent(10), "no description")
			}

			swagger = fmt.Sprintf("%s%scontent:\n", swagger, indent(10))
			swagger = fmt.Sprintf("%s%sapplication/json:\n", swagger, indent(12))
			swagger = fmt.Sprintf("%s%sschema:\n", swagger, indent(14))
			swagger = fmt.Sprintf("%s%stype: object\n", swagger, indent(16))
			swagger = fmt.Sprintf("%s%sproperties:\n", swagger, indent(16))

			if ret.StatusCode == "200" {
				jsonString := []byte(ret.JSON)
				var jsonMap map[string]interface{}
				err = json.Unmarshal(jsonString, &jsonMap)
				if err != nil {
					panic(err)
				}

				yamlBytes, err := yaml.Marshal(jsonMap)
				if err != nil {
					panic(err)
				}

				// fmt.Println(string(y))
				// fmt.Println("============================")

				yamlLines := strings.Split(string(yamlBytes), "\n")
				var arrInd uint
				for ln, yamlLine := range yamlLines {
					yamlLineTokens := strings.Split(yamlLine, ":")
					var extInd uint = 0
					if ln >= 1 && strings.HasSuffix(strings.Trim(yamlLines[ln-1], " "), ":") {
						extInd = 2
					}

					if (len(yamlLineTokens) == 1 && !util.IsEmptyOrWhitespace(yamlLineTokens[0])) ||
						(len(yamlLineTokens) == 2 && util.IsEmptyOrWhitespace(yamlLineTokens[1])) {

						yamlLineTokens[0] = strings.Replace(yamlLineTokens[0], "@", "", -1)
						if len(yamlLines) > ln && strings.Contains(yamlLines[ln+1], "- ") { // array
							swagger = fmt.Sprintf("%s%s%s:\n", swagger, indent(extInd+18), yamlLineTokens[0])
							swagger = fmt.Sprintf("%s%stype: array\n%sitems:\n%stype: object\n%sproperties:\n",
								swagger, indent(extInd+22), indent(extInd+22), indent(extInd+24), indent(extInd+24))
						} else {
							swagger = fmt.Sprintf("%s%s%s:\n", swagger, indent(extInd+18), yamlLineTokens[0])
							swagger = fmt.Sprintf("%s%stype: object\n", swagger, indent(extInd+20))
							swagger = fmt.Sprintf("%s%sproperties:\n", swagger, indent(extInd+20))
						}
					} else if len(yamlLineTokens) == 2 {
						if strings.Contains(yamlLineTokens[0], "- ") { // array
							yamlLineTokens[0] = strings.Replace(yamlLineTokens[0], "- ", "  ", 1)
							arrInd = util.CountLeadingSpaces(yamlLineTokens[0])
						}

						ind := util.CountLeadingSpaces(yamlLineTokens[0])
						if ind > 0 && ind == arrInd {
							ind += 2
						} else {
							arrInd = 0
						}

						if ln >= 1 && strings.HasPrefix(strings.TrimLeft(yamlLines[ln-1], " "), "'@") {
							swagger = strings.TrimSuffix(swagger, "properties:\n")
							swagger = strings.TrimRight(swagger, " ")
							swagger = strings.TrimSuffix(swagger, "type: object\n")
							swagger = strings.TrimRight(swagger, " ")
							swagger = fmt.Sprintf("%s%stype: object\n", swagger, indent(ind+20))
							swagger = fmt.Sprintf("%s%sadditionalProperties:\n", swagger, indent(ind+20))
							swagger = fmt.Sprintf("%s%stype: object\n", swagger, indent(ind+22))
						} else {
							exp := indent(ind+18) + yamlLineTokens[0]
							swagger = fmt.Sprintf("%s%s:\n", swagger, exp)
							swagger = fmt.Sprintf("%s%stype: %s\n",
								swagger, indent(util.CountLeadingSpaces(exp)+2),
								util.GoTypeToSwagger(strings.TrimLeft(yamlLineTokens[1], " ")))
						}
					}
				}
			} else {
				swagger = fmt.Sprintf("%s%sError:\n", swagger, indent(18))
				swagger = fmt.Sprintf("%s%srequired:\n", swagger, indent(20))
				swagger = fmt.Sprintf("%s%s- message\n", swagger, indent(22))
				swagger = fmt.Sprintf("%s%sproperties:\n", swagger, indent(20))
				swagger = fmt.Sprintf("%s%smessage:\n", swagger, indent(22))
				swagger = fmt.Sprintf("%s%stype: string\n", swagger, indent(24))
			}
		}
		// fmt.Println("============================")
		// fmt.Println("============================")
	}

	fmt.Println(swagger)
}

func indent(count uint) string {
	if count == 0 {
		return ""
	} else {
		return fmt.Sprintf("%s ", indent(count-1))
	}
}

type Field struct {
	Name        string
	Type        string
	TypeDef     string
	IsPrimitive bool
	IsArray     bool
	Attr        string
	JSONName    string
	RawVal      string
	Val         string
}

func getFieldsFromReturn(desc *Descriptor, returnStatement string, packages map[string]*ast.Package) []Field {
	responseFields := getResponseFields(returnStatement, packages)
	for fi, field := range responseFields {
		src, err := os.ReadFile(desc.Handler.HandlerPath)
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
				if x.Pos() < desc.Handler.HandlerFuncPos || x.End() > desc.Handler.HandlerFuncEnd {
					break
				}

				for li, lhs := range x.Lhs {
					start := lhs.Pos() - 1
					end := lhs.End() - 1
					if string(src[start:end]) != field.RawVal {
						continue
					}

					start = x.Rhs[li].Pos() - 1
					end = x.Rhs[li].End() - 1
					rhs := string(src[start:end])
					if !strings.Contains(rhs, "{") {
						continue
					}

					if strings.HasPrefix(field.Type, "[") {
						responseFields[fi].IsArray = true
						responseFields[fi].Type = util.GetStringAfter(field.Type, "]")
						_, structBody := findDeclPath(packages, fmt.Sprintf("type %s struct", responseFields[fi].Type))
						responseFields[fi].TypeDef = structBody
					} else {
						// TODO: Add something similar to above (array) for Map
						responseFields[fi].Type = strings.TrimLeft(util.GetStringBefore(rhs, "{"), "&")
						_, structBody := findDeclPath(packages, fmt.Sprintf("type %s struct", responseFields[fi].Type))
						responseFields[fi].TypeDef = structBody
					}
				}
			}

			return true
		})
		if ok, _ := util.IsPrimitiveType(responseFields[fi].Type); len(responseFields[fi].TypeDef) == 0 && !ok {
			responseFields[fi].Type = field.RawVal
			_, structBody := findDeclPath(packages, fmt.Sprintf("type %s struct", responseFields[fi].Type))
			responseFields[fi].TypeDef = structBody
		}
	}

	return responseFields
}

func getResponseFields(returnStatement string, packages map[string]*ast.Package) []Field {
	rawResponse := strings.Trim(strings.TrimRight(util.GetStringAfter(returnStatement, ","), ")"), " ")
	if strings.Contains(rawResponse, "{") {
		structName := util.GetStringBefore(rawResponse, "{")
		if strings.Contains(structName, ".") {
			structName = util.GetStringAfter(structName, ".")
		}

		_, b := findDeclPath(packages, fmt.Sprintf("type %s struct", structName))
		structLines := strings.Split(b, "\n")
		responseLines := strings.Split(strings.TrimRight(util.GetStringAfter(rawResponse, "{"), "}"), ",")
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
					primitiveType, _ := util.IsPrimitiveType(slTokens[1])
					r = &Field{
						Name:        slTokens[0],
						Type:        slTokens[1],
						IsPrimitive: primitiveType,
						Attr:        slTokens[2],
						JSONName:    util.GetStringInBetween(slTokens[2], "json:\"", "\""),
						RawVal:      strings.TrimRight(rlTokens[1], "{")}
				}
			}

			if r == nil {
				primitiveType, _ := util.IsPrimitiveType(slTokens[1])
				r = &Field{
					Name:        slTokens[0],
					Type:        slTokens[1],
					IsPrimitive: primitiveType,
					Attr:        slTokens[2],
					JSONName:    util.GetStringInBetween(slTokens[2], "json:\"", "\""),
					RawVal:      strings.Fields(responseLines[j-1])[0],
				}
			}

			if r.IsPrimitive {
				if strings.Contains(r.RawVal, "http.") {
					if v, ok := util.HTTPStatusCodes[strings.TrimLeft(r.RawVal, "http.")]; ok {
						r.Val = strconv.Itoa(v)
					}
				} else {
					r.Val = r.RawVal
				}
			}

			processedResponse = append(processedResponse, *r)
		}

		return processedResponse
	}
	return nil
}

type requestInputType string

const (
	ReqInpQueryParam = "QueryParam"
	ReqInpHeader     = "Request().Header.Get"
	EchoContextGet   = "*.Get"
)

func getQueryParams(desc *Descriptor, packages map[string]*ast.Package) map[string]string {
	return getRequestInputs(desc, packages, ReqInpQueryParam)
}

func getRequestHeaders(desc *Descriptor, packages map[string]*ast.Package) map[string]string {
	return getRequestInputs(desc, packages, ReqInpHeader)
}

func getFromContext(desc *Descriptor, packages map[string]*ast.Package) map[string]string {
	return getRequestInputs(desc, packages, EchoContextGet)
}

func getRequestInputs(desc *Descriptor, packages map[string]*ast.Package, inputType requestInputType) map[string]string {
	src, err := os.ReadFile(desc.Handler.HandlerPath)
	if err != nil {
		log.Fatal(err)
	}

	f, err := getFile(src, 0)
	if err != nil {
		log.Fatal(err)
	}

	ctxArgName := ""
	result := make(map[string]string)
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Pos() >= desc.Handler.HandlerFuncPos &&
				x.End() <= desc.Handler.HandlerFuncEnd {
				start := x.Pos() - 1
				end := x.End() - 1
				funcLines := strings.Split(strings.Trim(string(src[start:end]), "\n"), "\n")
				for _, fl := range funcLines {
					fl = strings.Trim(fl, " ")
					if strings.Contains(fl, "return func(") && strings.HasSuffix(fl, "echo.Context) error {") {
						ctxArgName = strings.Trim(util.GetStringInBetween(fl, "return func(", "echo.Context) error {"), " ")
					}
				}
			}
		case *ast.CallExpr:
			if x.Pos() < desc.Handler.HandlerFuncPos || x.End() > desc.Handler.HandlerFuncEnd {
				break
			}

			start := x.Pos() - 1
			end := x.End() - 1
			inputTypeStr := string(inputType)
			if strings.Contains(inputTypeStr, "*") {
				inputTypeStr = strings.Replace(inputTypeStr, "*", ctxArgName, 1)
			}

			if !strings.Contains(string(src[start:end]), inputTypeStr) {
				break
			}

			rawQueryParam := util.GetStringInBetween(string(src[start:end]), inputTypeStr+"(", ")")
			if strings.HasPrefix(rawQueryParam, `"`) && strings.HasSuffix(rawQueryParam, `"`) {
				result[util.GetStringInBetween(rawQueryParam, `"`, `"`)] = "string"
			} else {
				if strings.Contains(rawQueryParam, ".") {
					tokens := strings.Split(rawQueryParam, ".")
					rawQueryParam = tokens[len(tokens)-1]
				}

				_, decl := findDeclPath(packages, rawQueryParam)
				if !strings.HasPrefix(decl, "const (") { // TODO: extend for other declaration types
					break
				}

				declLines := strings.Split(strings.Trim(util.GetStringInBetween(decl, "const (", ")"), "\n"), "\n")
				for _, dl := range declLines {
					tokens := strings.Split(dl, "=")
					if !strings.Contains(tokens[0], rawQueryParam) {
						continue
					}

					result[util.GetStringInBetween(strings.Trim(tokens[1], " "), `"`, `"`)] = "string"
				}
			}
		}

		return true
	})

	return result
}

func tryGetErrorMsg(src string) string {
	msg := ""
	if strings.Contains(src, `fmt.Errorf("`) {
		msg = util.GetStringInBetween(src, `fmt.Errorf("`, `"`)
	} else if strings.Contains(src, `errors.New("`) {
		msg = util.GetStringInBetween(src, `errors.New("`, `"`)
	}
	return strings.Replace(msg, ":", "&#58;", -1)
}

func getHttpStatusCodeFromReturn(returnValue string) string {
	statusCode := util.GetStringInBetween(returnValue, ".JSON(", ",")
	code, ok := util.HTTPStatusCodes[strings.TrimLeft(statusCode, "http.")]
	if ok {
		return strconv.Itoa(code)
	} else {
		code, ok = util.HTTPStatusCodes[util.GetStringInBetween(returnValue, "http.", ",")]
		if ok {
			return strconv.Itoa(code)
		} else {
			return statusCode
		}
	}
}

func fillReturnsOfEachHandler(descriptors []*Descriptor) {
	for i, desc := range descriptors {
		handlerSrc, err := os.ReadFile(descriptors[i].Handler.HandlerPath)
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
				if !funcFound && x.Name.Name == descriptors[i].Handler.HandlerFuncName {
					funcFound = true
					firstFuncDecl = x
					desc.Handler.HandlerFuncPos = x.Pos()
					desc.Handler.HandlerFuncEnd = x.End()
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
					desc.Handler.RawReturns = append(desc.Handler.RawReturns, string(handlerSrc[start:end]))
				}
			}

			return true
		})
	}
}

func fillHeadersOfMiddleware(descriptors []*Descriptor, packages map[string]*ast.Package) {
	for i, _ := range descriptors {
		for _, middleware := range descriptors[i].Middlewares {
			middlewareSrc, err := os.ReadFile(middleware.Path)
			if err != nil {
				log.Fatal(err)
			}

			f, err := getFile(middlewareSrc, 0)
			if err != nil {
				log.Fatal(err)
			}

			var firstFuncDecl *ast.FuncDecl
			var funcFound bool
			var middlewareFuncPos, middlewareFuncEnd token.Pos
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					if !funcFound && x.Name.Name == middleware.Name {
						funcFound = true
						firstFuncDecl = x
						middlewareFuncPos = x.Pos()
						middlewareFuncEnd = x.End()
					}
				}

				return true
			})

			f, err = getFile(middlewareSrc, 0)
			if err != nil {
				log.Fatal(err)
			}

			// var rawReturns []string
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.ReturnStmt:
					if len(x.Results) < 1 || x.Pos() <= firstFuncDecl.Pos() || x.End() >= firstFuncDecl.End() {
						break
					}

					start := x.Results[0].Pos() - 1
					end := x.Results[0].End() - 1
					retStr := string(middlewareSrc[start:end])
					if !strings.HasPrefix(retStr, "func(ctx echo.Context) error") {
						break
					}

					funcLines := strings.Split(retStr, "\n")
					for _, l := range funcLines {
						if !strings.Contains(l, "Header.Get(") {
							continue
						}

						headerDecl := util.GetStringInBetween(l, "Header.Get(", ")")
						headerDecl = util.GetStringAfter(headerDecl, ".")
						_, headerVal := findDeclPath(packages, headerDecl)
						headerVal = util.GetStringInBetween(headerVal, fmt.Sprintf("%s = ", headerDecl), "\n")
						descriptors[i].Headers = append(descriptors[i].Headers, Header{Decl: headerDecl, Value: headerVal})
					}
				}

				return true
			})
		}
	}
}

func fillHandlerPath(descriptors []*Descriptor, packages map[string]*ast.Package) {
	for _, desc := range descriptors {
		desc.Handler.HandlerPath, _ = findDeclPath(packages, desc.Handler.HandlerFuncName)
	}
}

func fillMiddlewaresPath(descriptors []*Descriptor, packages map[string]*ast.Package) {
	for i, desc := range descriptors {
		for j, middleware := range desc.Middlewares {
			descriptors[i].Middlewares[j].Path, _ = findDeclPath(packages, middleware.Name)
		}
	}
}

func findDeclPath(packages map[string]*ast.Package, name string) (string, string) {
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

			var declBody string
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.GenDecl:
					start := x.Pos() - 1
					end := x.End() - 1
					b := string(src[start:end])
					if declBody == "" &&
						(strings.Contains(b, fmt.Sprintf(" %s(", name)) ||
							(strings.HasPrefix(name, "type") && strings.Contains(b, name)) ||
							(strings.HasPrefix(b, "const ") && strings.Contains(b, name)) ||
							(strings.HasPrefix(b, "var ") && strings.Contains(b, name))) {
						declBody = b

						return true
					}
				}

				return true
			})
			if len(declBody) > 0 {
				return path, declBody
			}

			for _, d := range file.Decls {
				switch x := d.(type) {
				case *ast.FuncDecl:
					if name == x.Name.String() {
						return path, ""
					}
				}
			}
		}
	}

	return "", ""
}

func getJSONBody(packages map[string]*ast.Package, field Field, isArray bool) string {
	var extInd uint
	var extOpen string
	if isArray {
		extInd = 2
		extOpen = " ["
	}

	json := fmt.Sprintf("%s\"%s\": %s", indent(extInd+2), field.JSONName, extOpen)
	if field.IsPrimitive == true {
		json = fmt.Sprintf("%s\"%s\",\n", json, strings.TrimLeft(field.Type, "*"))
	} else if strings.Contains(field.TypeDef, " struct {") {
		innerJson := fmt.Sprintf("\n%s{\n", indent(extInd+2))
		structLines := strings.Split(util.GetStringAfter(field.TypeDef, "{"), "\n")
		for _, sl := range structLines {
			slTokens := strings.Fields(sl)
			if len(slTokens) != 3 || !strings.Contains(slTokens[2], "json") {
				continue
			}

			if ok, _ := util.IsPrimitiveType(slTokens[1]); ok {
				innerJson = fmt.Sprintf("%s%s\"%s\": \"%s\",\n",
					innerJson, indent(extInd+4), util.GetStringInBetween(slTokens[2],
						"`json:\"", "\"`"), strings.TrimLeft(slTokens[1], "*"))
			} else if strings.HasPrefix(slTokens[1], "map[") {
				mapTypes := strings.Split(util.GetStringAfter(slTokens[1], "["), "]")
				if len(mapTypes) != 2 {
					continue
				}

				innerJson = fmt.Sprintf("%s%s\"@%s\": {\n",
					innerJson, indent(extInd+4), util.GetStringInBetween(slTokens[2],
						"`json:\"", "\"`"))
				innerJson = fmt.Sprintf("%s%s\"%s\": \"%s\"\n",
					innerJson, indent(extInd+6), mapTypes[0], mapTypes[1])
				innerJson = fmt.Sprintf("%s%s},\n",
					innerJson, indent(extInd+4))
			} else if strings.HasPrefix(slTokens[1], "[") { // array or slice
				t := util.GetStringAfter(slTokens[1], "]")
				if len(t) == 0 {
					continue
				}

				jsonName := util.GetStringInBetween(slTokens[2], "`json:\"", "\"`")
				_, structBody := findDeclPath(packages, fmt.Sprintf("type %s struct", t))
				if ok, _ := util.IsPrimitiveType(t); !ok {
					innerJson = fmt.Sprintf("%s%s", innerJson,
						getJSONBody(packages, Field{
							IsPrimitive: false,
							JSONName:    jsonName,
							Type:        t,
							TypeDef:     structBody,
						}, true))
				}
			}
		}

		innerJson = strings.TrimRight(innerJson, ",\n")
		innerJson = fmt.Sprintf("%s\n%s}\n", innerJson, indent(extInd+2))
		if isArray {
			innerJson = fmt.Sprintf("%s%s]\n", innerJson, indent(extInd))
		}
		json = fmt.Sprintf("%s%s", json, innerJson)
		// fmt.Println("--->", json)
	}

	return json
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

				descriptors[i].Handler.URL = fmt.Sprintf("%s%s",
					util.GetStringInBetween(assignmentRight, "\"", "\""), descriptors[i].Handler.URL)

				if strings.Contains(assignmentRight, ".Group(") {
					f, err := getFile(src, 0)
					if err != nil {
						log.Fatal(err)
					}

					ast.Inspect(f, findAssignment(src, lastFuncDecl, util.GetStringBefore(assignmentRight, ".Group("), descriptors, i))
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
					TagEnd:       c.End(),
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

func fillHandler(descriptors []*Descriptor) {
	for i := range descriptors {
		src, lastCallExpr, lastFuncDecl, lastSelectorExprStr := inspect(descriptors, i, descriptors[i].TagEnd, false)
		f, err := getFile(src, 0)
		if err != nil {
			log.Fatal(err)
		}

		f, err = getFile(src, 0)
		if err != nil {
			log.Fatal(err)
		}

		for j, arg := range lastCallExpr.Args {
			var argFound bool
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.BasicLit:
					if x.Pos() < arg.Pos() || argFound {
						break
					}

					argFound = true
					switch j {
					case 0:
						descriptors[i].Handler.URL = util.GetStringInBetween(x.Value, "\"", "\"")
					case 1:
						descriptors[i].Handler.HandlerFuncName = util.GetStringInBetween(x.Value, "\"", "\"")
					}
				}
				return true
			})
		}

		f, err = getFile(src, 0)
		if err != nil {
			log.Fatal(err)
		}

		ast.Inspect(f, findAssignment(src, &lastFuncDecl, lastSelectorExprStr, descriptors, i))
	}
}

func fillMiddlewares(descriptors []*Descriptor) {
	for i := range descriptors {
		src, lastCallExpr, lastFuncDecl, lastSelectorExprStr := inspect(descriptors, i, descriptors[i].middlewaresStart, true)
		f, err := getFile(src, 0)
		if err != nil {
			log.Fatal(err)
		}

		for j, arg := range lastCallExpr.Args {
			var argFound bool
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.BasicLit:
					if x.Pos() < arg.Pos() || argFound {
						break
					}

					argFound = true
					if j != 1 {
						break
					}

					mw := util.GetStringInBetween(x.Value, "\"", "\"")
					if !strings.HasSuffix(descriptors[i].Handler.URL, mw) {
						descriptors[i].Middlewares = append(descriptors[i].Middlewares, Middleware{Name: mw})
					}
				}

				return true
			})
		}

		f, err = getFile(src, 0)
		if err != nil {
			log.Fatal(err)
		}

		ast.Inspect(f, findAssignment(src, &lastFuncDecl, lastSelectorExprStr, descriptors, i))
	}
}

func inspect(descriptors []*Descriptor, descIndex int, pos token.Pos, isMiddleware bool) (src []byte, lastCallExpr ast.CallExpr, lastFuncDecl ast.FuncDecl, lastSelectorExprStr string) {
	var err error
	src, err = os.ReadFile(descriptors[descIndex].TaggedInPath)
	if err != nil {
		log.Fatal(err)
	}

	f, err := getFile(src, 0)
	if err != nil {
		log.Fatal(err)
	}

	var lastSelectorExpr *ast.SelectorExpr
	var callerFound, selectorFound bool
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if !callerFound {
				lastFuncDecl = *x
			}
		case *ast.SelectorExpr:
			if x.Pos() > pos && !callerFound {
				callerFound = true
				lastSelectorExpr = x

				start := lastSelectorExpr.X.Pos() - 1
				end := lastSelectorExpr.X.End() - 1
				lastSelectorExprStr = string(src[start:end])

				if !isMiddleware {
					start = lastSelectorExpr.Sel.Pos() - 1
					end = lastSelectorExpr.Sel.End() - 1
					descriptors[descIndex].Handler.Method = string(src[start:end])
				}
			}
		case *ast.CallExpr:
			if x.Pos() > pos && !selectorFound {
				selectorFound = true
				descriptors[descIndex].middlewaresStart = x.Pos()
				lastCallExpr = *x
			}
		}
		return true
	})
	return
}

func getFile(src []byte, p parser.Mode) (*ast.File, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", src, p)
	return f, err
}
