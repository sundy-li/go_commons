package assert

import (
	"fmt"
	"strings"
	"testing"
)

type FakeTester struct {
	str   string
	count int
}

func (f *FakeTester) Errorf(format string, args ...interface{}) {
	f.str = fmt.Sprintf(format, args...)
	f.count++
}

func Foo() string { return "foo" }

func TestValidAssert(t *testing.T) {
	var f FakeTester

	Equals(&f, Foo(), "foo")

	if f.count != 0 {
		t.Errorf("assert equals error; called %d times", f.count)
	}

	// should contain the line that caused the error
	if f.str != "" {
		t.Errorf("assert equals error; got [%v]", f)
	}
}

func TestFaltyAssert(t *testing.T) {
	var f FakeTester

	Equals(&f, Foo(), "bar")

	if f.count != 1 {
		t.Errorf("assert equals error; called %d times", f.count)
	}

	// line
	if !strings.Contains(f.str, `37`) {
		t.Errorf("assert equals error; got [%v]", f)
	}

	// file name
	if !strings.Contains(f.str, `assert_test.go`) {
		t.Errorf("assert equals error; got [%v]", f)
	}

	// should contain the line that caused the error
	if !strings.Contains(f.str, `Equals(&f, Foo(), "bar")`) {
		t.Errorf("assert equals error; got [%v]", f)
	}

	// expected
	if !strings.Contains(f.str, `expected: "bar"`) {
		t.Errorf("assert equals error; got [%v]", f)
	}

	// got
	if !strings.Contains(f.str, `got: "foo"`) {
		t.Errorf("assert equals error; got [%v]", f)
	}

	// should contain no newlines
	if strings.Contains(f.str, "FakeTester") || strings.Contains(f.str, "f.count") {
		t.Errorf("assert equals error; got [%v]", f)
	}
}

func TestTrue(t *testing.T) {
	{
		var f FakeTester

		True(&f, falsifier())

		Equals(t, f.count, 1)
		Equals(t, strings.Contains(f.str, `78`), true)
		Equals(t, strings.Contains(f.str, `assert_test.go`), true)
		Equals(t, strings.Contains(f.str, `True(&f, falsifier())`), true)
		Equals(t, strings.Contains(f.str, "FakeTester") || strings.Contains(f.str, "f.count"), false)
	}

	{
		var f FakeTester

		True(&f, truthifier())

		Equals(t, f.count, 0)
		Equals(t, f.str, "")
	}
}

func falsifier() bool  { return false }
func truthifier() bool { return true }

func TestFalse(t *testing.T) {
	{
		var f FakeTester
		False(&f, truthifier())

		Equals(t, f.count, 1)
		Equals(t, strings.Contains(f.str, `103`), true)
		Equals(t, strings.Contains(f.str, `assert_test.go`), true)
		Equals(t, strings.Contains(f.str, `False(&f, truthifier())`), true)
		Equals(t, strings.Contains(f.str, "FakeTester") || strings.Contains(f.str, "f.count"), false)
	}

	{
		var f FakeTester
		False(&f, falsifier())

		Equals(t, f.count, 0)
		Equals(t, f.str, "")
	}
}

func TestNotEqual(t *testing.T) {
	{
		var f FakeTester
		NotEquals(&f, Foo(), "foo")

		Equals(t, f.count, 1)
		True(t, strings.Contains(f.str, `124`))
		True(t, strings.Contains(f.str, `assert_test.go`))
		True(t, strings.Contains(f.str, `NotEquals(&f, Foo(), "foo")`))
		False(t, strings.Contains(f.str, "FakeTester") || strings.Contains(f.str, "f.count"))
	}

	{
		var f FakeTester
		NotEquals(&f, Foo(), "bar")

		Equals(t, f.count, 0)
		Equals(t, f.str, "")
	}
}

func TestDeepEqual(t *testing.T) {
	nums := func() []int { return []int{1, 2, 3} }
	{
		var f FakeTester
		DeepEquals(&f, nums(), []int{1, 2, 4})

		Equals(t, f.count, 1)
		True(t, strings.Contains(f.str, `146`))
		True(t, strings.Contains(f.str, `assert_test.go`))
		True(t, strings.Contains(f.str, `DeepEquals(&f, nums(), []int{1, 2, 4})`))
		False(t, strings.Contains(f.str, "FakeTester") || strings.Contains(f.str, "f.count"))

		True(t, strings.Contains(f.str, fmt.Sprintf(`expected: %#v`, []int{1, 2, 4})))
		True(t, strings.Contains(f.str, fmt.Sprintf(`got: %#v`, []int{1, 2, 3})))
	}

	{
		var f FakeTester
		DeepEquals(&f, nums(), []int{1, 2, 3})

		Equals(t, f.count, 0)
		Equals(t, f.str, "")
	}
}

func customEqualsForTesting(t TestDriver, got, expected string) {
	if got != expected {
		Errorf(t, "expected: %#v\n     got: %#v", expected, got)
	}
}

func TestErrorf(t *testing.T) {
	{
		var f FakeTester
		customEqualsForTesting(&f, Foo(), "bar")

		Equals(t, f.count, 1)
		True(t, strings.Contains(f.str, `176`))
		True(t, strings.Contains(f.str, `assert_test.go`))
		True(t, strings.Contains(f.str, `customEqualsForTesting(&f, Foo(), "bar")`))
		False(t, strings.Contains(f.str, "FakeTester") || strings.Contains(f.str, "f.count"))

		True(t, strings.Contains(f.str, `expected: "bar"`))
		True(t, strings.Contains(f.str, `got: "foo"`))
	}

	{
		var f FakeTester
		customEqualsForTesting(&f, Foo(), "foo")

		Equals(t, f.count, 0)
		Equals(t, f.str, "")
	}
}

func TestStringContains(t *testing.T) {
	{
		var f FakeTester
		StringContains(&f, "foo", "hi")

		Equals(t, f.count, 1)
		True(t, strings.Contains(f.str, `200`))
		True(t, strings.Contains(f.str, `assert_test.go`))
		True(t, strings.Contains(f.str, `StringContains(&f, "foo", "hi")`))
		False(t, strings.Contains(f.str, "FakeTester") || strings.Contains(f.str, "f.count"))

		True(t, strings.Contains(f.str, `expected: "foo"`))
		True(t, strings.Contains(f.str, `to contain: "hi"`))
	}

	{
		var f FakeTester
		StringContains(&f, "foo", "oo")

		Equals(t, f.count, 0)
		Equals(t, f.str, "")
	}
}

func TestShowOff(t *testing.T) {
	return

	Equals(t, "foo", "bar")
	NotEquals(t, "foo", "foo")
	DeepEquals(t, "foo", "bar")
	True(t, "foo" == "bar")
	False(t, "foo" == "foo")
	customEqualsForTesting(t, "1", "2")
	StringContains(t, "flub", "ber")
}
