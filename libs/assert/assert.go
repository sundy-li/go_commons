package assert

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"path"
	"reflect"
	"runtime"
	"strings"
)

type TestDriver interface {
	Errorf(format string, args ...interface{})
}

func auxiliaryInfo(extraStacks int) (filename string, line int, code string) {
	_, file, line, _ := runtime.Caller(3 + extraStacks)
	buf, _ := ioutil.ReadFile(file)
	filename = path.Base(file)
	code = strings.TrimSpace(strings.Split(string(buf), "\n")[line-1])
	return
}

func printError(t TestDriver, extraStacks int, format string, args ...interface{}) {
	filename, line, code := auxiliaryInfo(extraStacks)

	var buf bytes.Buffer
	fmt.Fprintf(NewWriter(&buf, Red), "\n%v:%d\n%s", filename, line, code)
	fmt.Fprintf(NewWriter(&buf, Purple), format, args...)

	t.Errorf(buf.String())
}

func Equals(t TestDriver, got, expected interface{}) {
	if got != expected {
		printError(t, 0, "\n\n\texpected: %#v\n\t     got: %#v", expected, got)
	}
}
func AlmostEquals(t TestDriver, got, expected float64) {
	if !(math.Abs(got-expected) < 0.00001) {
		printError(t, 0, "\n\n\texpected: %#v\n\t     got: %#v", expected, got)
	}
}
func Nil(t TestDriver, got interface{}) {
	if got != nil {
		printError(t, 0, "\n\n\texpected: %#v\n\t     got: %#v", nil, got)
	}
}
func NoNil(t TestDriver, got interface{}) {
	if got == nil {
		printError(t, 0, "\n\n\texpected: no nil\n\t     got: %#v", got)
	}
}
func DeepEquals(t TestDriver, got, expected interface{}) {
	if !reflect.DeepEqual(got, expected) {
		printError(t, 0, "\n\n\texpected: %#v\n\t     got: %#v", expected, got)
	}
}

func NotEquals(t TestDriver, got, expected interface{}) {
	if got == expected {
		printError(t, 0, "\n\n\tunexpectedly got: %#v", got)
	}
}

func True(t TestDriver, got bool) {
	if got != true {
		printError(t, 0, "")
	}
}

func False(t TestDriver, got bool) {
	if got != false {
		printError(t, 0, "")
	}
}

func Errorf(t TestDriver, format string, args ...interface{}) {
	format = "\t" + strings.Replace(format, "\n", "\n\t", -1) // indent every line once
	printError(t, 1, "\n\n"+format, args...)
}

func StringContains(t TestDriver, full, fragment string) {
	if !strings.Contains(full, fragment) {
		Errorf(t, "   expected: %#v\n to contain: %#v", full, fragment)
	}
}
