package cytracker

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"net/url"
)

func TestParsing(t *testing.T) {
	Convey("Value parsing", t, func() {
		k := "key"
		k2 := "key2"
		k3 := "key3"
		Convey("Boolean", func() {
			values := new(Values)
			values.Values = url.Values{}
			values.Add(k, "true")
			values.Add(k2, "false")
			values.Add(k3, "trufalse")
			Convey("Ok", func() {
				v, err := values.GetBool(k)
				So(err, ShouldBeNil)
				So(v, ShouldEqual, true)
				v2, err := values.GetBool(k2)
				So(err, ShouldBeNil)
				So(v2, ShouldEqual, false)
			})
			Convey("Not found", func() {
				_, err := values.GetBool("kek")
				So(err, ShouldNotBeNil)
			})
			Convey("Bad value", func() {
				_, err := values.GetBool(k3)
				So(err, ShouldNotBeNil)
			})
		})
		Convey("Int", func() {
			values := new(Values)
			values.Values = url.Values{}
			values.Add(k, "32123")
			values.Add(k2, "-32123")
			values.Add(k3, "trufalse")
			Convey("Ok", func() {
				v, err := values.GetInt(k)
				So(err, ShouldBeNil)
				So(v, ShouldEqual, 32123)
				v2, err := values.GetInt(k2)
				So(err, ShouldBeNil)
				So(v2, ShouldEqual, -32123)
			})
			Convey("Not found", func() {
				_, err := values.GetInt("kek")
				So(err, ShouldNotBeNil)
			})
			Convey("Bad value", func() {
				_, err := values.GetInt(k3)
				So(err, ShouldNotBeNil)
			})
		})
		Convey("Uint65", func() {
			values := new(Values)
			values.Values = url.Values{}
			values.Add(k, "32123234")
			values.Add(k2, "-32123234")
			values.Add(k3, "3dvd312")
			Convey("Ok", func() {
				v, err := values.GetInt(k)
				So(err, ShouldBeNil)
				So(v, ShouldEqual, 32123234)
				v2, err := values.GetInt(k2)
				So(err, ShouldBeNil)
				So(v2, ShouldEqual, -32123234)
			})
			Convey("Not found", func() {
				_, err := values.GetInt("kek")
				So(err, ShouldNotBeNil)
			})
			Convey("Bad value", func() {
				_, err := values.GetInt(k3)
				So(err, ShouldNotBeNil)
			})
		})
	})
}
