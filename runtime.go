package spinix

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/geojson/geo"
	"github.com/tidwall/geojson/geometry"
	"github.com/uber/h3-go"
)

type invoker interface {
	invoke(ctx context.Context, d *Device, ref reference) (Match, error)
}

type reference struct {
	rules      Rules
	objects    Objects
	geospatial Geospatial
	devices    Devices
}

type Match struct {
	Ok       bool
	Left     Decl
	Right    Decl
	Operator Token
	Pos      Pos
}

type Decl struct {
	Keyword Token
	Refs    []string
}

func defaultRefs() reference {
	return reference{
		devices:    NewDevices(),
		objects:    NewObjects(),
		geospatial: DefaultGeospatial(),
		rules:      NewRules(),
	}
}

type spec struct {
	nodes []invoker
	ops   []Token
	pos   Pos
}

func specFromString(s string) (*spec, error) {
	expr, err := ParseSpec(s)
	if err != nil {
		return nil, err
	}
	return exprToSpec(expr)
}

func (s *spec) invoke(ctx context.Context, d *Device, r reference) (matches []Match, err error) {
	if len(s.nodes) == 0 {
		return
	}

	if len(s.nodes) == 1 {
		match, err := s.nodes[0].invoke(ctx, d, r)
		if err != nil {
			return nil, err
		}
		if match.Ok {
			return []Match{match}, nil
		}
	}

	var (
		index int
		op    Token
		ok    bool
	)

	for index < len(s.nodes) {
		var right Match
		node := s.nodes[index]
		if node == nil {
			continue
		}

		if index > 0 {
			if !ok && op == AND {
				if index < len(s.ops) {
					op = s.ops[index]
				}
				index++
				continue
			}
		}

		right, err = s.nodes[index].invoke(ctx, d, r)
		if err != nil {
			return nil, err
		}
		if index < len(s.ops) {
			op = s.ops[index]
		}
		switch op {
		case AND:
			if index == 0 {
				ok = right.Ok
			} else {
				ok = ok && right.Ok
			}
		case OR:
			if index == 0 {
				ok = right.Ok
			} else {
				ok = ok || right.Ok
			}
		}
		if right.Ok {
			if matches == nil {
				matches = make([]Match, 0, len(s.nodes))
			}
			matches = append(matches, right)
		}
		index++
	}
	return
}

func walkExpr(
	expr Expr,
	exprFunc func(a, b Expr, op Token) error,
	opFunc func(tok Token),
) (Expr, error) {
	if expr == nil {
		return nil, fmt.Errorf("spinix/runtime: expression is nil")
	}
	switch n := expr.(type) {
	case *ParenExpr:
		return walkExpr(n.Expr, exprFunc, opFunc)
	case *BinaryExpr:
		lhs, err := walkExpr(n.LHS, exprFunc, opFunc)
		if err != nil {
			return nil, err
		}
		if n.Op == AND || n.Op == OR {
			opFunc(n.Op)
		}
		rhs, err := walkExpr(n.RHS, exprFunc, opFunc)
		if err != nil {
			return nil, err
		}
		_, lbe := lhs.(*BinaryExpr)
		_, rbe := rhs.(*BinaryExpr)
		if !lbe && !rbe {
			if err := exprFunc(lhs, rhs, n.Op); err != nil {
				return nil, err
			}
		}
	}
	return expr, nil
}

func exprToSpec(e Expr) (*spec, error) {
	s := &spec{
		ops:   make([]Token, 0, 2),
		nodes: make([]invoker, 0, 2),
	}
	_, err := walkExpr(e,
		func(a, b Expr, op Token) error {
			node, err := makeOp(a, b, op)
			if err != nil {
				return err
			}
			s.nodes = append(s.nodes, node)
			return nil
		}, func(tok Token) {
			s.ops = append(s.ops, tok)
		})
	if err != nil {
		return nil, err
	}
	if len(s.nodes) == 0 {
		return nil, fmt.Errorf("spinix/runtime: specification not defined")
	}
	if len(s.nodes)-1 != len(s.ops) {
		return nil, fmt.Errorf("spinix/runtime: invalid specification %s", e)
	}
	return s, nil
}

func makeOp(left, right Expr, op Token) (invoker, error) {
	switch op {
	case NEAR:
		return e2near(left, right)
	case RANGE:
		return e2range(left, right)
	case IN:
		return e2in(left, right)
	}
	return nil, fmt.Errorf("spinix/runtime: unknown operator %v %v %v",
		left, op, right)
}

func e2in(left, right Expr) (invoker, error) {
	lhs, ok := left.(*IdentLit)
	if !ok {
		return nil, fmt.Errorf("spinix/runtime: invalid expr: %s IN %s",
			left, right)
	}

	switch lhs.Kind {
	case MODEL, BRAND, OWNER, IMEI, FUELLEVEL, PRESSURE, LUMINOSITY, HUMIDITY, TEMPERATURE,
		BATTERY_CHARGE, STATUS, SPEED, YEAR, MONTH, WEEK, DAY, HOUR, DATE, DATETIME:
	default:
		return nil, fmt.Errorf("spinix/runtime: invalid IN operator got %s, expected [%s], pos=%d",
			lhs, tok2Str(MODEL, BRAND, OWNER, IMEI, FUELLEVEL, PRESSURE,
				LUMINOSITY, HUMIDITY, TEMPERATURE, BATTERY_CHARGE, STATUS, SPEED,
				YEAR, MONTH, WEEK, DAY, HOUR, DATE, DATETIME), lhs.Pos)
	}

	rhs, ok := right.(*ListLit)
	if !ok || rhs.Kind != ILLEGAL {
		return nil, fmt.Errorf("spinix/runtime: invalid expr: %s IN %s",
			left, right)
	}

	switch rhs.Typ {
	case INT:
		op := inIntOp{
			keyword: lhs.Kind,
			pos:     rhs.Pos,
			values:  make([]int, len(rhs.Items)),
		}
		for i := 0; i < len(rhs.Items); i++ {
			n := rhs.Items[i].(*IntLit)
			op.values[i] = n.Value
		}
		return op, nil
	case FLOAT:
		op := inFloatOp{
			keyword: lhs.Kind,
			pos:     rhs.Pos,
			values:  make([]float64, len(rhs.Items)),
		}
		for i := 0; i < len(rhs.Items); i++ {
			n := rhs.Items[i].(*FloatLit)
			op.values[i] = n.Value
		}
		return op, nil
	case STRING, IDENT:
		op := inStringOp{
			keyword: lhs.Kind,
			pos:     rhs.Pos,
			values:  make([]string, len(rhs.Items)),
		}
		for i := 0; i < len(rhs.Items); i++ {
			n := rhs.Items[i].(*StringLit)
			op.values[i] = n.Value
		}
		return op, nil
	}
	return nil, fmt.Errorf("spinix/runtime: invalid expr: %s IN %s",
		left, right)
}

func e2range(left, right Expr) (invoker, error) {
	// int -> int
	// float -> float
	// time -> time
	// dateTime -> dateTime
	isPropKeyword := func(op Token) bool {
		switch op {
		case TIME, FUELLEVEL, PRESSURE, LUMINOSITY, HUMIDITY, TEMPERATURE,
			BATTERY_CHARGE, STATUS, SPEED, YEAR, MONTH, WEEK, DAY, HOUR, DATE, DATETIME:
			return true
		}
		return false
	}

	switch lhs := left.(type) {
	case *IdentLit:
		if !isPropKeyword(lhs.Kind) {
			break
		}
		switch rhs := right.(type) {
		case *ListLit:
			if rhs.Kind != RANGE {
				break
			}
			switch rhs.Typ {
			case INT:
				switch lhs.Kind {
				case FUELLEVEL, PRESSURE, LUMINOSITY, HUMIDITY, TEMPERATURE,
					BATTERY_CHARGE, STATUS, SPEED, YEAR, MONTH, WEEK, DAY, HOUR:
				default:
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator got %s, expected [%s], pos=%d",
						lhs, tok2Str(FUELLEVEL, PRESSURE, LUMINOSITY, HUMIDITY, TEMPERATURE,
							BATTERY_CHARGE, STATUS, SPEED, YEAR, MONTH, WEEK, DAY, HOUR), rhs.Pos)
				}
				begin, ok := rhs.Items[0].(*IntLit)
				if !ok {
					break
				}
				end, ok := rhs.Items[1].(*IntLit)
				if !ok {
					break
				}
				if begin.Value > end.Value {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator left %s operand is greater than right %s, pos=%d",
						begin, end, rhs.Pos)
				}
				if begin.Value == end.Value {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator left %s and right %s operands are equal, pos=%d",
						begin, end, rhs.Pos)
				}
				return rangeIntOp{
					begin:   begin.Value,
					end:     end.Value,
					pos:     rhs.Pos,
					keyword: lhs.Kind,
				}, nil
			case FLOAT:
				switch lhs.Kind {
				case FUELLEVEL, PRESSURE, LUMINOSITY, HUMIDITY, TEMPERATURE,
					BATTERY_CHARGE, STATUS, SPEED, YEAR, MONTH, WEEK, DAY, HOUR:
				default:
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator got %s, expected [%s], pos=%d",
						lhs, strings.Join([]string{
							FUELLEVEL.String(),
							PRESSURE.String(),
							LUMINOSITY.String(),
							HUMIDITY.String(),
							TEMPERATURE.String(),
							BATTERY_CHARGE.String(),
							STATUS.String(),
							SPEED.String(),
							YEAR.String(),
							MONTH.String(),
							WEEK.String(),
							DAY.String(),
							HOUR.String()}, ","), rhs.Pos)
				}
				begin, ok := rhs.Items[0].(*FloatLit)
				if !ok {
					break
				}
				end, ok := rhs.Items[1].(*FloatLit)
				if !ok {
					break
				}
				if begin.Value > end.Value {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator left %s operand is greater than right %s, pos=%d",
						begin, end, rhs.Pos)
				}
				if begin.Value == end.Value {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator left %s and right %s operands are equal, pos=%d",
						begin, end, rhs.Pos)
				}
				return rangeFloatOp{
					begin:   begin.Value,
					end:     end.Value,
					pos:     rhs.Pos,
					keyword: lhs.Kind,
				}, nil
			case TIME:
				if lhs.Kind != TIME {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator got %s, expected %s, pos=%d",
						lhs.Kind, TIME, rhs.Pos)
				}
				begin := rhs.Items[0].(*TimeLit)
				end := rhs.Items[1].(*TimeLit)
				if begin.Hour < 0 || begin.Hour > 23 {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator got %d, expected hour >= 0 and hour < 24, pos=%d",
						begin.Hour, rhs.Pos)
				}
				if begin.Minute < 0 || begin.Minute > 59 {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator got %d, expected minutes >= 0 and minutes < 59, pos=%d",
						begin.Minute, rhs.Pos)
				}
				if end.Hour < 0 || end.Hour > 23 {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator got %d, expected hour >= 0 and hour < 24, pos=%d",
						end.Hour, rhs.Pos)
				}
				if end.Minute < 0 || end.Minute > 59 {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator got %d, expected minutes >= 0 and minutes < 59, pos=%d",
						end.Minute, rhs.Pos)
				}
				return rangeTimeOp{
					begin:   timeVal{h: begin.Hour, m: begin.Minute},
					end:     timeVal{h: end.Hour, m: end.Minute},
					pos:     rhs.Pos,
					keyword: lhs.Kind,
				}, nil
			case STRING:
				if lhs.Kind != DATE && lhs.Kind != DATETIME {
					break
				}
				begin, ok := rhs.Items[0].(*StringLit)
				if !ok {
					break
				}
				end, ok := rhs.Items[1].(*StringLit)
				if !ok {
					break
				}
				var pattern string
				switch lhs.Kind {
				case DATE:
					pattern = "2006-01-02"
				default:
					pattern = time.RFC3339
				}

				beginValue := strings.ReplaceAll(begin.Value, `"`, "")
				left, err := time.Parse(pattern, beginValue)
				if err != nil {
					break
				}
				endValue := strings.ReplaceAll(end.Value, `"`, "")
				right, err := time.Parse(pattern, endValue)
				if err != nil {
					break
				}

				// left == right
				if lhs.Kind == DATETIME && left.Unix() == right.Unix() {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator left %s and right %s operands are equal, pos=%d",
						left, right, rhs.Pos)
				}
				// left > right
				if lhs.Kind == DATETIME && left.Unix() > right.Unix() {
					return nil, fmt.Errorf("spinix/runtime: invalid RANGE operator left %s operand is greater than right %s, pos=%d",
						left, right, rhs.Pos)
				}
				return rangeDateTimeOp{
					keyword: lhs.Kind,
					begin:   left,
					end:     right,
					pos:     rhs.Pos,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("spinix/runtime: invalid expr: %s RANGE %s",
		left, right)
}

func e2equal(left, right Expr) (invoker, error) {
	return nil, nil
}

func e2near(left, right Expr) (invoker, error) {
	var node nearOp
	// device -> object
	// device -> devices
	// object -> device
	// devices -> device
	switch lhs := left.(type) {
	case *DeviceLit:
		node.device = lhs
		switch rhs := right.(type) {
		case *ObjectLit:
			node.object = rhs
			node.pos = rhs.Pos
		case *DevicesLit:
			node.devices = rhs
			node.pos = rhs.Pos
		case *DeviceLit:
			node.other = rhs
			node.pos = rhs.Pos
			switch node.device.Unit {
			case DistanceMeters, DistanceKilometers:
			default:
				node.device.Unit = DistanceMeters
				node.device.Value = 1000
			}
		default:
			return node, fmt.Errorf("spinix/runtime: invalid spec => %s NEAR %s, pos=%v",
				lhs, rhs, lhs.Pos)
		}
	case *ObjectLit:
		node.object = lhs
		switch rhs := right.(type) {
		case *DeviceLit:
			node.device = rhs
			node.pos = rhs.Pos
		default:
			return node, fmt.Errorf("spinix/runtime: invalid spec => %s NEAR %s, pos=%v",
				lhs, rhs, lhs.Pos)
		}
	case *DevicesLit:
		node.devices = lhs
		switch rhs := right.(type) {
		case *DeviceLit:
			node.device = rhs
			node.pos = rhs.Pos
		default:
			return node, fmt.Errorf("spinix/runtime: invalid spec => %s NEAR %s, pos=%v",
				lhs, rhs, lhs.Pos)
		}
	default:
		return node, fmt.Errorf("spinix/runtime: invalid spec => %s NEAR %s",
			left, right)
	}

	switch node.device.Unit {
	case DistanceMeters, DistanceKilometers:
		if node.device.Value <= 0 {
			return node, fmt.Errorf("spinix/runtime: invalid distance value in spec => %s, operator=%v, pos=%v",
				node.device, NEAR, node.device.Pos)
		}
	}
	if node.devices != nil && node.object != nil && node.other != nil {
		return node, fmt.Errorf("spinix/runtime: invalid spec => %s NEAR %s",
			left, right)
	}
	return node, nil
}

type nearOp struct {
	// left
	device *DeviceLit

	// right
	object  *ObjectLit
	devices *DevicesLit
	other   *DeviceLit

	pos Pos
}

func (n nearOp) invoke(ctx context.Context, d *Device, ref reference) (match Match, err error) {
	// device
	var (
		meters       float64
		deviceRadius *geometry.Poly
		devicePoint  geometry.Point
	)

	switch n.device.Unit {
	case DistanceKilometers:
		if n.device.Value > 0 {
			meters = n.device.Value * 1000
		}
	case DistanceMeters:
		meters = n.device.Value
	}

	switch n.device.Kind {
	case RADIUS, BBOX:
		// circle or rect
		ring := makeRadiusRing(d.Latitude, d.Longitude, meters, 16)
		deviceRadius = &geometry.Poly{Exterior: ring}
	default:
		// point
		devicePoint = geometry.Point{X: d.Latitude, Y: d.Longitude}
	}

	// device -> objects(polygon, circle, rect, ...)
	if n.object != nil {
		for i := 0; i < len(n.object.Ref); i++ {
			objectID := n.object.Ref[i]
			obj, err := ref.objects.Lookup(ctx, objectID)
			if err != nil {
				if errors.Is(err, ErrObjectNotFound) {
					continue
				}
				return match, err
			}
			if obj == nil {
				return match, nil
			}
			switch n.device.Kind {
			case RADIUS:
				if deviceRadius != nil && obj.Spatial().IntersectsPoly(deviceRadius) {
					match.Ok = true
					if match.Right.Refs == nil {
						match.Right.Refs = make([]string, 0, len(n.object.Ref))
					}
					match.Right.Refs = append(match.Right.Refs, objectID)
				}
			case BBOX:
				if deviceRadius != nil && obj.Spatial().IntersectsRect(deviceRadius.Rect()) {
					match.Ok = true
					if match.Right.Refs == nil {
						match.Right.Refs = make([]string, 0, len(n.object.Ref))
					}
					match.Right.Refs = append(match.Right.Refs, objectID)
				}
			default:
				if obj.Spatial().IntersectsPoint(devicePoint) {
					match.Ok = true
					if match.Right.Refs == nil {
						match.Right.Refs = make([]string, 0, len(n.object.Ref))
					}
					match.Right.Refs = append(match.Right.Refs, objectID)
				}
			}
		}
		if match.Ok {
			match.Left.Keyword = DEVICE
			match.Left.Refs = []string{d.IMEI}
			match.Operator = NEAR
			match.Pos = n.pos
			match.Right.Keyword = OBJECTS
		}
		return match, nil
	}

	// device -> devices
	if n.devices != nil {
		var otherDeviceMeters float64
		switch n.devices.Unit {
		case DistanceKilometers:
			if n.devices.Value > 0 {
				otherDeviceMeters = n.devices.Value * 1000
			}
		case DistanceMeters:
			otherDeviceMeters = n.devices.Value
		}
		var otherDeviceRadius *geometry.Poly
		var otherDevicePoint geometry.Point
		for _, otherDeviceID := range n.devices.Ref {
			otherDevice, err := ref.devices.Lookup(ctx, otherDeviceID)
			if err != nil {
				if errors.Is(err, ErrDeviceNotFound) {
					continue
				}
				return match, err
			}
			switch n.devices.Kind {
			case RADIUS, BBOX:
				// circle
				ring := makeRadiusRing(
					otherDevice.Latitude,
					otherDevice.Longitude,
					otherDeviceMeters, 16)
				otherDeviceRadius = &geometry.Poly{Exterior: ring}
				switch n.devices.Kind {
				case RADIUS:
					if deviceRadius != nil && otherDeviceRadius.IntersectsPoly(deviceRadius) {
						match.Ok = true
						if match.Right.Refs == nil {
							match.Right.Refs = make([]string, 0, len(n.devices.Ref))
						}
						match.Right.Refs = append(match.Right.Refs, otherDeviceID)
					}
				case BBOX:
					if deviceRadius != nil && otherDeviceRadius.IntersectsRect(deviceRadius.Rect()) {
						match.Ok = true
						if match.Right.Refs == nil {
							match.Right.Refs = make([]string, 0, len(n.devices.Ref))
						}
						match.Right.Refs = append(match.Right.Refs, otherDeviceID)
					}
				default:
					if otherDeviceRadius.IntersectsPoint(devicePoint) {
						match.Ok = true
						if match.Right.Refs == nil {
							match.Right.Refs = make([]string, 0, len(n.devices.Ref))
						}
						match.Right.Refs = append(match.Right.Refs, otherDeviceID)
					}
				}
			default:
				// point
				otherDevicePoint = geometry.Point{
					X: otherDevice.Latitude,
					Y: otherDevice.Longitude,
				}
				if otherDevicePoint.IntersectsPoint(devicePoint) {
					match.Ok = true
					if match.Right.Refs == nil {
						match.Right.Refs = make([]string, 0, len(n.devices.Ref))
					}
					match.Right.Refs = append(match.Right.Refs, otherDeviceID)
				}
			}
		}
		if match.Ok {
			match.Left.Keyword = DEVICE
			match.Left.Refs = []string{d.IMEI}
			match.Operator = NEAR
			match.Pos = n.pos
			match.Right.Keyword = DEVICES
		}
		return match, nil
	}

	// device -> device
	if n.other != nil {
		err := ref.devices.Nearby(ctx, d.Latitude, d.Longitude, meters,
			func(ctx context.Context, other *Device) error {
				if d.IMEI == other.IMEI {
					return nil
				}
				match.Ok = true
				if match.Right.Refs == nil {
					match.Right.Refs = make([]string, 0, 8)
				}
				match.Right.Refs = append(match.Right.Refs, other.IMEI)
				return nil
			})
		if err != nil {
			return match, err
		}
		if match.Ok {
			match.Left.Keyword = DEVICE
			match.Left.Refs = []string{d.IMEI}
			match.Operator = NEAR
			match.Pos = n.pos
			match.Right.Keyword = DEVICE
		}
		return match, nil
	}

	return
}

type rangeDateTimeOp struct {
	keyword Token
	begin   time.Time
	end     time.Time
	pos     Pos
}

func (n rangeDateTimeOp) invoke(ctx context.Context, d *Device, ref reference) (match Match, err error) {
	return
}

type timeVal struct {
	h, m int
}

type rangeTimeOp struct {
	keyword Token
	begin   timeVal
	end     timeVal
	pos     Pos
}

func (n rangeTimeOp) invoke(ctx context.Context, d *Device, ref reference) (match Match, err error) {
	return
}

type rangeIntOp struct {
	keyword Token
	begin   int
	end     int
	pos     Pos
}

func (n rangeIntOp) invoke(ctx context.Context, d *Device, ref reference) (match Match, err error) {
	return
}

type rangeFloatOp struct {
	keyword Token
	begin   float64
	end     float64
	pos     Pos
}

func (n rangeFloatOp) invoke(ctx context.Context, d *Device, ref reference) (match Match, err error) {
	return
}

type inFloatOp struct {
	keyword Token
	pos     Pos
	values  []float64
}

func (n inFloatOp) invoke(ctx context.Context, d *Device, ref reference) (match Match, err error) {
	return
}

type inIntOp struct {
	keyword Token
	pos     Pos
	values  []int
}

func (n inIntOp) invoke(ctx context.Context, d *Device, ref reference) (match Match, err error) {
	return
}

type inStringOp struct {
	keyword Token
	pos     Pos
	values  []string
}

func (n inStringOp) invoke(ctx context.Context, d *Device, ref reference) (match Match, err error) {
	return
}

type equalStrOp struct {
	left  string
	right string
	pos   Pos
}

func (n equalStrOp) invoke(ctx context.Context, d *Device, ref reference) (match Match, err error) {
	return
}

func isSmallRadius(meters float64) bool {
	return meters < maxRadiusInMeters
}

func getSteps(meters float64) (steps int) {
	steps = 16
	if !isSmallRadius(meters) {
		steps = 8
	}
	return
}

func getLevel(meters float64) (level int) {
	level = smallLevel
	if !isSmallRadius(meters) {
		level = largeLevel
	}
	return
}

func cover(meters float64, level int, points []geometry.Point) []h3.H3Index {
	smallSearchRadius := isSmallRadius(meters)
	steps := getSteps(meters)
	visits := make(map[h3.H3Index]struct{})
	res := make([]h3.H3Index, 0, 2)
	half := steps / 2
	for i, p := range points {
		idx := h3.FromGeo(h3.GeoCoord{Latitude: p.X, Longitude: p.Y}, level)
		_, visit := visits[idx]
		if !visit {
			res = append(res, idx)
			visits[idx] = struct{}{}
		}
		if smallSearchRadius {
			continue
		}
		if i <= half {
			p1 := points[i+half]
			b := geo.BearingTo(p.X, p.Y, p1.X, p1.Y)
			d := geo.DistanceTo(p.X, p.Y, p1.X, p1.Y)
			s := d / float64(steps)
			for i := float64(0); i <= d; i += s {
				lat, lng := geo.DestinationPoint(p.X, p.Y, i, b)
				idx := h3.FromGeo(h3.GeoCoord{Latitude: lat, Longitude: lng}, level)
				_, visit := visits[idx]
				if !visit {
					res = append(res, idx)
					visits[idx] = struct{}{}
				}
			}
		}
	}
	return res
}

func newCircle(lat, lng float64, meters float64, steps int) (points []geometry.Point, bbox geometry.Rect) {
	meters = geo.NormalizeDistance(meters)
	points = make([]geometry.Point, 0, steps+1)
	for i := 0; i < steps; i++ {
		b := (i * -360) / steps
		lat, lng := geo.DestinationPoint(lat, lng, meters, float64(b))
		point := geometry.Point{X: lat, Y: lng}
		points = append(points, point)
		if i == 0 {
			bbox.Min = point
			bbox.Max = point
		} else {
			if point.X < bbox.Min.X {
				bbox.Min.X = point.X
			} else if point.X > bbox.Max.X {
				bbox.Max.X = point.X
			}
			if point.Y < bbox.Min.Y {
				bbox.Min.Y = point.Y
			} else if points[i].Y > bbox.Max.Y {
				bbox.Max.Y = points[i].Y
			}
		}
	}
	points = append(points, points[0])
	return
}

func contains(p geometry.Point, points []geometry.Point) bool {
	for i := 0; i < len(points); i++ {
		var seg geometry.Segment
		seg.A = points[i]
		if i == len(points)-1 {
			seg.B = points[0]
		} else {
			seg.B = points[i+1]
		}
		res := seg.Raycast(p)
		if res.In {
			return true
		}
	}
	return false
}

type radiusRing struct {
	rect   geometry.Rect
	points []geometry.Point
}

func makeRadiusRing(lat, lng float64, meters float64, steps int) radiusRing {
	rr := radiusRing{}
	rr.points, rr.rect = newCircle(lat, lng, meters, steps)
	return rr
}

func (rr radiusRing) Index() interface{} {
	return nil
}

func (rr radiusRing) Clockwise() bool {
	return true
}

func (rr radiusRing) Move(_, _ float64) geometry.Series {
	return rr
}

func (rr radiusRing) Empty() bool {
	return false
}

func (rr radiusRing) Valid() bool {
	return true
}

func (rr radiusRing) Rect() geometry.Rect {
	return rr.rect
}

func (rr radiusRing) Convex() bool {
	return true
}

func (rr radiusRing) Closed() bool {
	return true
}

func (rr radiusRing) NumPoints() int {
	return len(rr.points)
}

func (rr radiusRing) PointAt(index int) geometry.Point {
	return rr.points[index]
}

func (rr radiusRing) Search(rect geometry.Rect, iter func(seg geometry.Segment, idx int) bool) {
	n := rr.NumSegments()
	for i := 0; i < n; i++ {
		seg := rr.SegmentAt(i)
		if seg.Rect().IntersectsRect(rect) {
			if !iter(seg, i) {
				return
			}
		}
	}
}

func (rr radiusRing) NumSegments() int {
	if len(rr.points) < 3 {
		return 0
	}
	if rr.points[len(rr.points)-1] == rr.points[0] {
		return len(rr.points) - 1
	}
	return len(rr.points)
}

func (rr radiusRing) SegmentAt(index int) geometry.Segment {
	var seg geometry.Segment
	seg.A = rr.points[index]
	if index == len(rr.points)-1 {
		seg.B = rr.points[0]
	} else {
		seg.B = rr.points[index+1]
	}
	return seg
}

func hasMatches(m []Match) bool {
	return len(m) > 0
}

func tok2Str(tokens ...Token) string {
	res := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		res = append(res, tok.String())
	}
	return strings.Join(res, ",")
}