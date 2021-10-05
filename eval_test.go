package georule

import (
	"testing"
)

//func TestEvalFunc(t *testing.T) {
//	expr := rule(t, "intersectsPoly(@id) and intersectsPoly(@id2)")
//	res, err := eval(expr, &Device{}, &State{})
//	if err != nil {
//		t.Fatal(err)
//	}
//	_ = res
//}
//
//func TestEval(t *testing.T) {
//	testCases := []struct {
//		name   string
//		device *Device
//		expr   []Expr
//		isErr  bool
//		want   string
//	}{
//		{
//			name: "device.speed",
//			device: &Device{
//				Speed: 40,
//			},
//			expr: []Expr{
//				rule(t, "{device.speed} >= 0 AND {device.speed} <= 60"),
//				rule(t, "{device.speed} > 0 AND {device.speed} < 60"),
//				rule(t, "{device.speed} != 0 AND {device.speed} > 30"),
//			},
//			want: "true",
//		},
//
//		{
//			name: "device.battery",
//			device: &Device{
//				BatteryCharge: 15,
//			},
//			expr: []Expr{
//				rule(t, "{device.battery} >= 0 AND {device.battery} <= 60"),
//				rule(t, "{device.battery} > 0 AND {device.battery} < 60"),
//			},
//			want: "true",
//		},
//
//		{
//			name: "device.temperature",
//			device: &Device{
//				Temperature: 85,
//			},
//			expr: []Expr{
//				rule(t, "{device.temperature} >= 50 AND {device.temperature} <= 90"),
//				rule(t, "{device.temperature} > 70 AND {device.temperature} < 86"),
//			},
//			want: "true",
//		},
//
//		{
//			name: "device.humidity",
//			device: &Device{
//				Speed:       50,
//				Temperature: 89,
//				Humidity:    78,
//			},
//			expr: []Expr{
//				rule(t, "{device.humidity} >= 50 AND {device.humidity} <= 90"),
//				rule(t, "{device.humidity} > 70 AND {device.humidity} < 86"),
//			},
//			want: "true",
//		},
//
//		{
//			name: "device.luminosity",
//			device: &Device{
//				Luminosity: 3,
//			},
//			expr: []Expr{
//				rule(t, "{device.luminosity} >= 0 AND {device.luminosity} <= 9"),
//				rule(t, "{device.luminosity} > 0 AND {device.luminosity} < 5"),
//			},
//			want: "true",
//		},
//
//		{
//			name: "device.pressure",
//			device: &Device{
//				Pressure: 3,
//			},
//			expr: []Expr{
//				rule(t, "{device.pressure} >= 0 AND {device.pressure} <= 9"),
//				rule(t, "{device.pressure} > 0 AND {device.pressure} < 5"),
//			},
//			want: "true",
//		},
//
//		{
//			name: "device.fuellevel",
//			device: &Device{
//				FuelLevel: 3,
//			},
//			expr: []Expr{
//				rule(t, "{device.fuellevel} >= 0 AND {device.fuellevel} <= 9"),
//				rule(t, "{device.fuellevel} > 0 AND {device.fuellevel} < 5"),
//			},
//			want: "true",
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			for _, rule := range tc.expr {
//				res, err := eval(rule, tc.device, &State{}, nil, nil)
//				if tc.isErr {
//					if err == nil {
//						t.Fatalf("eval(%s) => got nil, expected non nil error", tc.expr)
//					} else {
//						return
//					}
//				} else if err != nil {
//					t.Fatal(err)
//				}
//				if res.String() != tc.want {
//					t.Fatalf("eval(%s) => %s, want %s", tc.expr, res, tc.want)
//				}
//			}
//		})
//	}
//}

func rule(t *testing.T, spec string) Expr {
	expr, err := ParseString(spec)
	if err != nil {
		t.Fatal(err)
	}
	return expr
}
