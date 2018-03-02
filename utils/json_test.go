package utils

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMergeMaps(t *testing.T) {

	Convey(
		"Give two struct A,B", t, func() {
			type A struct {
				A1 int
				A2 string
			}
			type B struct {
				B1 int
				B2 string `json:"b2"`
			}
			a := &A{1, "a"}
			b := &B{2, "b"}
			Convey("When merge a and b", func() {
				res := MergeMaps(a, b)
				Convey("The result should be", func() {
					So(res["A1"], ShouldEqual, 1)
					So(res["b2"], ShouldEqual, "b")
				})
			})

		})
}

func TestToMap(t *testing.T) {
	type A struct {
		A1 int
		A2 string
	}
	Convey("Test ToMap", t, func() {
		Convey("when convert struct", func() {
			a := &A{1, "a"}
			m, _ := ToMap(a, "json")
			So(m["A1"], ShouldEqual, 1)
			So(m["A2"], ShouldEqual, "a")
		})
	})
}

func TestUUID(t *testing.T) {
	Convey("Test UUID", t, func() {
		Convey("when convert struct", func() {
			aa := UUID()
			println(aa)
		})
	})
}
