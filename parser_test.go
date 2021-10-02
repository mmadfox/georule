package georule

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name  string
		rule  string
		isErr bool
		typ   Expr
	}{
		// success vars cases
		{
			name:  "parse {device.status}",
			rule:  `({device.status} == 1 OR {device.status} IN [2,4]) OR ({device.status} >= 0 AND {device.status} < 10)`,
			isErr: false,
			typ:   &BinaryExpr{},
		},

		{
			name:  "parse {device.speed} variable",
			rule:  `{device.speed} >= 0 AND {device.speed} <= 50`,
			isErr: false,
			typ:   &BinaryExpr{},
		},

		// success func cases
		{
			name: "parse fuellevel, pressure, luminosity, humidity, temperature, batteryCharge, speed rule",
			rule: `(
                       fuellevel(0, 10) OR fuellevel(10)
                   ) AND (
                       pressure(0, 10) OR pressure(40)
                   ) AND (
                       luminosity(0, 10) OR luminosity(300)
                   ) AND (
                       humidity(0, 40) OR humidity(50)  
                   ) AND (
                       temperature(0, 10) OR temperature(90)
                   ) AND (
                       batteryCharge(0, 10) OR batteryCharge(40)
                   ) AND (
                       speed(0, 10) OR speed(50)
                   )`,
			isErr: false,
			typ:   &BinaryExpr{},
		},

		{
			name: "parse distance rule",
			rule: `(
                        distanceLine(@line1) >= 3000 AND distance(@lin2) <= 9000
                   ) OR (
                        distancePoint(@polygon1) > 0 and distanceRect(@polygon2) < 10 and distanceRect(@rect1) == 400
                   )`,
			isErr: false,
			typ:   &BinaryExpr{},
		},

		{
			name: "parse within rule",
			rule: `(
                        within(@line) AND withinLine(@lin2, @line1, @line3)
                   ) OR (
                        withinPoint(@polygon1) and withinPoly(@polygon2) and withinRect(@rect1)
                   )`,
			isErr: false,
			typ:   &BinaryExpr{},
		},

		{
			name: "parse not within rule",
			rule: `(
                        not within(@line) AND not withinLine(@lin2, @line1, @line3)
                   ) OR (
                        not withinPoint(@polygon1) and not withinPoly(@polygon2) and not withinRect(@rect1)
                   )`,
			isErr: false,
			typ:   &BinaryExpr{},
		},

		{
			name: "parse intersects rule",
			rule: `(
                        intersects(@line) AND intersectsLine(@lin2, @line1, @line3)
                   ) OR (
                        intersectsPoint(@polygon1) and intersectsPoly(@polygon2) and intersectsRect(@rect1)
                   )`,
			isErr: false,
			typ:   &BinaryExpr{},
		},

		{
			name: "parse not intersects rule",
			rule: `(
                        not intersects(@line) AND not intersectsLine(@lin2, @line1, @line3)
                   ) OR (
                        not intersectsPoint(@polygon1) and not intersectsPoly(@polygon2) and not intersectsRect(@rect1)
                   )`,
			isErr: false,
			typ:   &BinaryExpr{},
		},

		{
			name:  "parse contains rule",
			rule:  "contains(@point, @line, @poly, @rect)",
			isErr: false,
			typ:   &CallExpr{},
		},

		{
			name:  "parse not contains rule",
			rule:  "not contains(@point, @line, @poly, @rect)",
			isErr: false,
			typ:   &CallExpr{},
		},

		{
			name:  "parse speed rule",
			rule:  "speed(0, 20) OR speed(20.3)",
			isErr: false,
			typ:   &BinaryExpr{},
		},

		// failure cases
		{
			name:  "parse invalid variable",
			rule:  `{somevar}`,
			isErr: true,
		},

		{
			name:  "parse invalid someFunc rule",
			rule:  `someFunc(@line)`,
			isErr: true,
		},

		{
			name:  "parse to long ident",
			rule:  fmt.Sprintf("intersectsLine(@%s)", strings.Repeat("s", 257)),
			isErr: true,
		},

		{
			name:  "parse exceeds the number of arguments",
			rule:  "speed(0, 20, 30)",
			isErr: true,
		},

		{
			name:  "parse without arguments",
			rule:  "emei() OR brand() OR owner()",
			isErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := ParseString(tc.rule)
			if tc.isErr {
				if err == nil {
					t.Fatalf("ParseString(%s) => got nil, expected non nil error", tc.rule)
				} else {
					return
				}
			}
			if expr == nil {
				t.Fatalf("ParseString(%s) => got expr nil, expected non nil expr", tc.rule)
			} else {
				have := reflect.TypeOf(expr).Elem().Name()
				want := reflect.TypeOf(tc.typ).Elem().Name()
				if have != want {
					t.Fatalf("ParseString(%s) => got %s, expected %s", tc.rule, have, want)
				}
			}
		})
	}
}
