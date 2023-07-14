package util

import (
	"strconv"
	"strings"
)

var HTTPStatusCodes = map[string]int{
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

func GetStringInBetween(str string, start string, end string) (result string) {
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

func GetStringBefore(value string, a string) string {
	pos := strings.Index(value, a)
	if pos == -1 {
		return ""
	}
	return value[0:pos]
}

func GetStringAfter(value string, a string) string {
	pos := strings.Index(value, a)
	if pos == -1 {
		return ""
	}
	return value[pos+1:]
}

func CountLeadingSpaces(str string) uint {
	return uint(len(str) - len(strings.TrimLeft(str, " ")))
}

func IsEmptyOrWhitespace(str string) bool {
	return len(str) == 0 || strings.Trim(str, " ") == ""
}

func IsPrimitiveType(t string) (bool, string) {
	types := map[string]string{
		"complex64":   "(0+0i)",
		"complex128":  "(0+0i)",
		"float32":     "0.0",
		"float64":     "0.0",
		"uint":        "0",
		"uint8":       "0",
		"uint16":      "0",
		"uint32":      "0",
		"uint64":      "0",
		"int":         "0",
		"int8":        "0",
		"int16":       "0",
		"int32":       "0",
		"int64":       "0",
		"uintptr":     "0",
		"error":       "nil",
		"bool":        "true",
		"string":      "\"string\"",
		"time.Time":   "\"time\"",
		"*complex64":  "(0+0i)",
		"*complex128": "(0+0i)",
		"*float32":    "0.0",
		"*float64":    "0.0",
		"*uint":       "0",
		"*uint8":      "0",
		"*uint16":     "0",
		"*uint32":     "0",
		"*uint64":     "0",
		"*int":        "0",
		"*int8":       "0",
		"*int16":      "0",
		"*int32":      "0",
		"*int64":      "0",
		"*uintptr":    "0",
		"*error":      "nil",
		"*bool":       "true",
		"*string":     "\"string\"",
		"*time.Time":  "\"time\"",
	}

	defaultValue, ok := types[t]
	return ok, defaultValue
}

func GoTypeToSwagger(t string) string {
	switch t {
	case "complex64",
		"complex128",
		"float32",
		"float64",
		"uint",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"int",
		"int8",
		"int16",
		"int32",
		"int64",
		"uintptr",
		"*complex64",
		"*complex128",
		"*float32",
		"*float64",
		"*uint",
		"*uint8",
		"*uint16",
		"*uint32",
		"*uint64",
		"*int",
		"*int8",
		"*int16",
		"*int32",
		"*int64",
		"*uintptr":
		return "number"
	case "bool",
		"*bool":
		return "boolean"
	case "string",
		"*string",
		"time.Time",
		"*time.Time":
		return "string"
	default:
		return ""
	}
}

func GetTypeByValue(value string) string {
	if _, err := strconv.Atoi(value); err == nil {
		return "int"
	} else if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "float"
	} else if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
		(strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		return "string"
	}

	return ""
}
