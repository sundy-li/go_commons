package crypto

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	eKey = "005e3a7b01ba94cc043d07e801ba94cc043d13a001ba94cc043d1b70ebb7c3b5"
	iKey = "005e3a7b01ba94cc043d3e9801ba94cc043d4a5001ba94cc043d4e380baa427f"
)

func TestDecode(t *testing.T) {
	decrypter, err := New(eKey, iKey)
	Convey("decode test", t, func() {
		Convey("new decrypter should be success", func() {
			So(err, ShouldBeNil)
		})

		oText := int64(1452616300)
		src := `Jn1WFirQqMKW4x849B9Uz1-bxpA=`
		text, err := decrypter.Decode(src)
		Convey("decode1 should be success", func() {
			So(err, ShouldBeNil)
			So(text, ShouldEqual, oText)
		})

		src = `Wf5-m4pgIFbulS2kaU38XhT62m0=`
		text, err = decrypter.Decode(src)
		Convey("decode2 should be success", func() {
			So(err, ShouldBeNil)
			So(text, ShouldEqual, oText)
		})
	})
}

func TestEncode(t *testing.T) {
	decrypter, err := New(eKey, iKey)
	Convey("encode and decode test", t, func() {
		Convey("new decrypter should be success", func() {
			So(err, ShouldBeNil)
		})

		oText := int64(1452616300)
		eText, err := decrypter.Encode(oText)
		Convey("encode should be success", func() {
			So(err, ShouldBeNil)
		})

		text, err := decrypter.Decode(eText)
		Convey("decode should be success", func() {
			So(err, ShouldBeNil)
			So(text, ShouldEqual, oText)
		})
	})
}

func BenchmarkEncode(b *testing.B) {
	decrypter, _ := New(eKey, iKey)
	oText := int64(14526163)
	for i := 0; i < b.N; i++ {
		decrypter.Encode(oText)
	}
}
