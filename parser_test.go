package spinix

import (
	"fmt"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	testCases := []struct {
		spec  string
		isErr bool
	}{
		// successfully
		{
			spec: `device INTERSECTS polygon(object) AND speed range [1 .. 40] { :center 42.9284788 72.2776118 }`,
		},
		{
			spec: `device :radius 1km intersects polygon(object) { :center 42.9284788 72.2776118 }`,
		},
		{
			spec: `devices(my) :radius 100m near devices(@) :radius 100m { :trigger every 10s }`,
		},
		{
			spec: `devices(my) :radius 100m near devices(other1, other2, other2) :radius 100m { :trigger every 10s }`,
		},
		{
			spec: `device :radius 100m near devices(other1, other2, other2) :radius 100m { :trigger every 10s }`,
		},
		{
			spec: `
                 status eq 1 OR 1 eq status 
                 {  
                    :radius 3km 
                    :center 42.4984338 -72.4265129 
                    :trigger every 10s 
                    :expire 10h 
                    :reset after 24h
                 }
`,
		},
		{spec: `device :radius 4km intersects polygon(poly) { :reset after 24h :trigger 25 times interval 10s }`},
		{spec: `device :radius 4km intersects polygon(poly) :trigger once :reset after 24h`},
		{spec: `device :radius 4km intersects polygon(poly) :reset after 24h :trigger every 10s`},
		{spec: `device :radius 4km intersects polygon(poly)`},
		{spec: `device intersects polygon(poly)`},
		{spec: `device :radius 4km in polygon(poly)`},
		{spec: `device :radius 4km nin polygon(poly)`},
		{spec: `status eq 1 OR 1 eq status`},
		{spec: `device near polygon(poly) :time duration 5m0s`},
		{spec: `device near polygon(poly) :time after 5m0s`},
		{spec: `circle(poly) :time duration 5s near device :radius 5km`},
		{spec: `devices(poly, id2) :bbox 300m near devices(poly, poly) :bbox 400m`},
		{spec: `device :radius 300m intersects devices(poly, poly) :radius 400m`},
		{spec: `speed range [1 .. 60]`},
		{spec: `speed nrange [1 .. 60]`},
		{spec: `temperature range [2.2 .. 10.8]`},
		{spec: `temperature gte 1 and temperature lt 40`},
		{spec: `pressure gte 1 and pressure lt 40`},
		{spec: `luminosity gte 1 and luminosity lt 40`},
		{spec: `battery range [0 .. 30]`},
		{spec: `fuelLevel range [0 .. 30]`},
		{spec: `status range [0 .. 30]`},
		{spec: `humidity range [0 .. 30]`},
		{spec: `imei in ["one", "two"]`},
		{spec: `year range [2022 .. 2023]`},
		{spec: `month range [1 .. 12]`},
		{spec: `week in [48, 49, 50] and week range [40 .. 52]`},
		{spec: `day range [1 .. 12]`},
		{spec: `time range [12:00 .. 23:00]`},
		{spec: `time gt 12:00 and time lt 15:00`},
		{spec: `time eq 19:21`},
		{spec: `datetime range ["2012-11-01T22:08:41+00:00" .. "2012-11-01T22:08:41+00:00"]`},
		{spec: `datetime gte "2012-11-01T22:08:41+00:00" and datetime lt "2012-11-01T22:08:41+00:00"`},
		{spec: `datetime in ["2012-11-01T22:08:41+00:00", "2012-11-01T22:08:41+00:00"]`},
		{spec: `device :radius 300m intersects line(@poly) and speed range [30 .. 120]`},
		{spec: `
             device :radius 300m intersects line(poly) 
             and speed range [30 .. 120] { :trigger 25 times interval 10s }`},
		{spec: `
             device :radius 300m intersects line(poly) 
             and speed range [30 .. 120] { :trigger every 10s }`},
		{spec: `
             device :radius 300m intersects line(poly) 
             and speed range [30 .. 120] { :trigger once }`},

		{spec: `device :radius 300m intersects line(poly) and speed range [30 .. 120]
			or (temperature gte 0 and temperature lt 400)`},

		{spec: `
             device :radius 300m intersects line(poly) 
             and speed range [30 .. 120] :trigger`}, // ignore properties :trigger

		// failure
		{spec: "", isErr: true},
		{spec: "some text", isErr: true},
		{spec: `devices(,,,) intersects circle()`, isErr: true},
		{spec: `devices("one") intersects circle()`, isErr: true},
		{spec: `circle() intersects device`, isErr: true},
		{spec: `circle intersects device`, isErr: true},
		{spec: `circle(....) intersects device`, isErr: true},
		{spec: `device near polygon(poly) :time duration h3s`, isErr: true},
		{spec: fmt.Sprintf(`device near polygon(@%s) :time duration h3s`, strings.Repeat("o", 128)), isErr: true},
		{spec: `device near polygon(poly) :time before 5m0s`, isErr: true},
		{spec: `device near polygon(poly) :time after`, isErr: true},
		{spec: `device :radius b0km`, isErr: true},
		{spec: `speed range [0x0 .. b0]`, isErr: true},
		{spec: `speed range [0x0 .. b0.0]`, isErr: true},
		{spec: `owner in []`, isErr: true},
		{spec: `brand in [1 .. 2, 1, 3]`, isErr: true},
		{spec: `model in [1 ... 2]`, isErr: true},
		{spec: `iemi in [1, 1.1, "one"]`, isErr: true},
		{spec: `owner in [1.1, "one", 1]`, isErr: true},
		{spec: `owner in ["one", 1.1, 1]`, isErr: true},
		{spec: `owner in [1.1, 1]`, isErr: true},
		{spec: `time gt 12: and time lt 15:00`, isErr: true},
		{spec: `datetime gte 2012-11-01T22:08:41+00:00 and datetime lt 2012-11-01T22:08:41+00:00`, isErr: true},
		{spec: `
             device :radius 300m intersects line(poly) 
             and speed range [30 .. 120] { :trigger every hhh }`, isErr: true},
		{spec: `
             device :radius 300m intersects line(poly) 
             and speed range [30 .. 120] { :trigger every 300s somelit }`, isErr: true},
		{spec: `
             device :radius 300m intersects line(poly) 
             and speed range [30 .. 120] { :trigger 0x0 times }`, isErr: true},
		{spec: `
             device :radius 300m intersects line(poly) 
             and speed range [30 .. 120] { :trigger 4 somelit }`, isErr: true},
		{spec: `
             device :radius 300m intersects line(poly) 
             and speed range [30 .. 120] { :trigger 4 times some }`, isErr: true},
		{spec: `
             device :radius 300m intersects line(poly) 
             and speed range [30 .. 120] { :trigger 4 times interval h4 }`, isErr: true},
		{spec: `
             device :radius 300m intersects line(@poly) 
             and speed range [30 .. 120] { :trigger 4 times interval 300s somelit }`, isErr: true},
	}
	for _, tc := range testCases {
		expr, err := ParseSpec(tc.spec)
		if err != nil {
			if tc.isErr {
				continue
			} else {
				t.Fatal(err)
			}
		} else if tc.isErr {
			t.Fatalf("ParseSpec(%s) => nil, want err", tc.spec)
		}
		if expr == nil {
			t.Fatalf("ParseSpec(%s) => nil, want Expr", tc.spec)
		}
	}
}
