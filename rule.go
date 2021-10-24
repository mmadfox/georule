package spinix

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/btree"

	"github.com/rs/xid"

	"github.com/tidwall/rtree"

	"github.com/uber/h3-go"

	"github.com/tidwall/geojson/geometry"
)

const (
	smallCellSize     = 2
	largeCellSize     = 0
	minRadiusInMeters = 500
	maxRadiusInMeters = 100000
)

type Rules interface {
	Walk(ctx context.Context, device *Device, fn WalkRuleFunc) error
	Insert(ctx context.Context, r *Rule) error
	Delete(ctx context.Context, ruleID string) error
	FindOne(ctx context.Context, ruleID string) (*Rule, error)
	Find(ctx context.Context, f RulesFilter) ([]*Rule, error)
}

type WalkRuleFunc func(ctx context.Context, rule *Rule, err error) error

type RulesFilter struct {
}

type Rule struct {
	ruleID         string
	owner          string
	name           string
	spec           *spec
	descr          string
	meters         float64
	bbox           geometry.Rect
	regionIDs      []h3.H3Index
	regionCellSize int
	circle         radiusRing
	center         geometry.Point
}

func (r *Rule) calc(meters float64) {
	steps := getSteps(meters)
	regionLevel := getLevel(meters)
	circle, bbox := newCircle(r.center.X, r.center.Y, meters, steps)
	regionIDs := cover(meters, regionLevel, circle)
	if len(r.regionIDs) != len(regionIDs) {
		r.regionIDs = make([]h3.H3Index, len(regionIDs))
	}
	r.circle = radiusRing{
		points: circle,
		rect:   bbox,
	}
	r.regionIDs = regionIDs
	r.bbox = bbox
	r.regionCellSize = regionLevel
	r.meters = meters
}

func (r *Rule) Circle() geometry.Series {
	return r.circle
}

func (r *Rule) Owner() string {
	return r.owner
}

func (r *Rule) Bounding() geometry.Rect {
	return r.bbox
}

func (r *Rule) Center() geometry.Point {
	return r.center
}

func (r *Rule) ID() string {
	return r.ruleID
}

func (r *Rule) Less(b btree.Item) bool {
	return r.ruleID < b.(*Rule).ruleID
}

func (r *Rule) RefIDs() (refs map[string]Token) {
	for _, n := range r.spec.nodes {
		nodeRef := n.refIDs()
		if nodeRef == nil {
			continue
		}
		if refs == nil {
			refs = make(map[string]Token)
		}
		for k, v := range nodeRef {
			refs[k] = v
		}
	}
	return refs
}

func (r *Rule) validate() error {
	if len(r.ruleID) == 0 {
		return fmt.Errorf("spinix/rule: id not specified")
	}
	if len(r.name) == 0 {
		return fmt.Errorf("spinix/rule: %s name not specified", r.ruleID)
	}
	if r.meters < minRadiusInMeters {
		return fmt.Errorf("spinix/rule: %s search radius is less than %d meters",
			r.ruleID, minRadiusInMeters)
	}
	if len(r.regionIDs) == 0 {
		return fmt.Errorf("spinix/rule: %s region not specified", r.ruleID)
	}
	return nil
}

func NewRule(
	name string,
	owner string,
	spec string,
	lat float64, lon float64,
	meters float64,
) (*Rule, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("spinix/rule: specification too short")
	}
	if len(spec) > 1024 {
		return nil, fmt.Errorf("spinix/rule: specification too long")
	}
	if len(name) == 0 {
		return nil, fmt.Errorf("spinix/rule: name too short")
	}
	if len(name) > 180 {
		return nil, fmt.Errorf("spinix/rule: name too long")
	}
	if meters < minRadiusInMeters {
		meters = minRadiusInMeters
	}
	if meters > 100000000 {
		meters = 100000000
	}

	nodes, err := specFromString(spec)
	if err != nil {
		return nil, err
	}

	rule := &Rule{
		ruleID: xid.New().String(),
		name:   name,
		owner:  owner,
		descr:  spec,
		spec:   nodes,
		center: geometry.Point{X: lat, Y: lon},
	}
	rule.calc(meters)
	return rule, nil

}

type RuleSnapshot struct {
	RuleID       string   `json:"ruleID"`
	Name         string   `json:"name"`
	Spec         string   `json:"specStr"`
	Latitude     float64  `json:"lat"`
	Longitude    float64  `json:"lon"`
	RadiusMeters float64  `json:"radiusMeters"`
	RegionIDs    []uint64 `json:"regionIDs"`
	RegionLevel  int      `json:"regionCellSize"`
}

func Snapshot(r *Rule) RuleSnapshot {
	snapshot := RuleSnapshot{
		RuleID:       r.ruleID,
		Name:         r.name,
		Spec:         r.descr,
		Latitude:     r.center.X,
		Longitude:    r.center.Y,
		RadiusMeters: r.meters,
		RegionLevel:  r.regionCellSize,
		RegionIDs:    make([]uint64, len(r.regionIDs)),
	}
	for i := 0; i < len(r.regionIDs); i++ {
		snapshot.RegionIDs[i] = uint64(r.regionIDs[i])
	}
	return snapshot
}

type Stats struct {
}

func NewMemoryRules() Rules {
	return &rules{
		smallRegionIndex: newSmallRegionIndex(),
		largeRegionIndex: newLargeRegionIndex(),
		ruleIndex:        newRuleIndex(),
	}
}

func (r *rules) Stats() (Stats, error) {
	return Stats{}, nil
}

func (r *rules) Walk(ctx context.Context, device *Device, fn WalkRuleFunc) (err error) {
	if err := r.walkSmallRegion(ctx, device, fn); err != nil {
		return err
	}
	return r.walkLargeRegion(ctx, device, fn)
}

func (r *rules) Insert(_ context.Context, rule *Rule) (err error) {
	switch rule.regionCellSize {
	case smallCellSize:
		err = r.insertToSmallRegion(rule)
	case largeCellSize:
		err = r.insertToLargeRegion(rule)
	default:
		err = fmt.Errorf("georule/rules: region level %d not defined", rule.regionCellSize)
	}
	if err == nil {
		r.ruleIndex.set(rule)
	}
	return
}

func (r *rules) Delete(_ context.Context, ruleID string) error {
	rule, err := r.ruleIndex.get(ruleID)
	if err != nil {
		return err
	}
	for _, regionID := range rule.regionIDs {
		switch rule.regionCellSize {
		case smallCellSize:
			region, ok := r.smallRegionIndex.find(regionID)
			if !ok {
				continue
			}
			region.delete(rule)
			if region.isEmpty() {
				r.smallRegionIndex.delete(regionID)
			}
		case largeCellSize:
			region, ok := r.largeRegionIndex.find(regionID)
			if !ok {
				continue
			}
			region.delete(rule)
			if region.isEmpty() {
				r.largeRegionIndex.delete(regionID)
			}
		}
	}
	r.ruleIndex.delete(ruleID)
	return nil
}

func (r *rules) Find(ctx context.Context, f RulesFilter) ([]*Rule, error) {
	return nil, nil
}

func (r *rules) FindOne(_ context.Context, ruleID string) (*Rule, error) {
	return r.ruleIndex.get(ruleID)
}

func (r *rules) insertToLargeRegion(rule *Rule) error {
	for _, regionID := range rule.regionIDs {
		r.largeRegionIndex.findOrCreate(regionID).insertRule(rule)
	}
	return nil
}

func (r *rules) insertToSmallRegion(rule *Rule) error {
	for _, regionID := range rule.regionIDs {
		r.smallRegionIndex.findOrCreate(regionID).insertRule(rule)
	}
	return nil
}

func (r *rules) walkSmallRegion(ctx context.Context, device *Device, fn WalkRuleFunc) error {
	cord := h3.GeoCoord{Latitude: device.Latitude, Longitude: device.Longitude}
	regionID := h3.FromGeo(cord, smallCellSize)
	region, ok := r.smallRegionIndex.find(regionID)
	if !ok {
		return nil
	}
	return region.walk(ctx, device, fn)
}

func (r *rules) walkLargeRegion(ctx context.Context, device *Device, fn WalkRuleFunc) error {
	cord := h3.GeoCoord{Latitude: device.Latitude, Longitude: device.Longitude}
	regionID := h3.FromGeo(cord, largeCellSize)
	region, ok := r.largeRegionIndex.find(regionID)
	if !ok {
		return nil
	}
	return region.walk(ctx, device, fn)
}

type rules struct {
	counter          uint64
	smallRegionIndex *smallRegionIndex
	largeRegionIndex *largeRegionIndex
	ruleIndex        rulesIndex
}

type largeRegionIndex struct {
	index map[h3.H3Index]*ruleLargeRegion
	mu    sync.RWMutex
}

func (i *largeRegionIndex) find(id h3.H3Index) (*ruleLargeRegion, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	region, ok := i.index[id]
	if !ok {
		return nil, false
	}
	return region, true
}

func (i *largeRegionIndex) delete(id h3.H3Index) {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.index, id)
}

func (i *largeRegionIndex) findOrCreate(id h3.H3Index) *ruleLargeRegion {
	i.mu.RLock()
	region, found := i.index[id]
	i.mu.RUnlock()
	if found {
		return region
	}
	region = newRuleLargeRegion(id)
	i.mu.Lock()
	i.index[id] = region
	i.mu.Unlock()
	return region
}

func newLargeRegionIndex() *largeRegionIndex {
	return &largeRegionIndex{
		index: make(map[h3.H3Index]*ruleLargeRegion),
	}
}

type smallRegionIndex struct {
	index map[h3.H3Index]*ruleSmallRegion
	mu    sync.RWMutex
}

func (i *smallRegionIndex) find(id h3.H3Index) (*ruleSmallRegion, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	region, ok := i.index[id]
	if !ok {
		return nil, false
	}
	return region, true
}

func (i *smallRegionIndex) delete(id h3.H3Index) {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.index, id)
}

func (i *smallRegionIndex) findOrCreate(id h3.H3Index) *ruleSmallRegion {
	i.mu.RLock()
	region, found := i.index[id]
	i.mu.RUnlock()
	if found {
		return region
	}
	region = newRuleSmallRegion(id)
	i.mu.Lock()
	i.index[id] = region
	i.mu.Unlock()
	return region
}

func newSmallRegionIndex() *smallRegionIndex {
	return &smallRegionIndex{
		index: make(map[h3.H3Index]*ruleSmallRegion),
	}
}

type rulesIndex []*ruleBucket

type ruleBucket struct {
	sync.RWMutex
	index map[string]*Rule
}

func newRuleIndex() rulesIndex {
	buckets := make([]*ruleBucket, numBucket)
	for i := 0; i < numBucket; i++ {
		buckets[i] = &ruleBucket{
			index: make(map[string]*Rule),
		}
	}
	return buckets
}

func (i rulesIndex) bucket(ruleID string) *ruleBucket {
	return i[bucket(ruleID, numBucket)]
}

func (i rulesIndex) set(rule *Rule) {
	bucket := i.bucket(rule.ruleID)
	bucket.Lock()
	bucket.index[rule.ruleID] = rule
	bucket.Unlock()
}

func (i rulesIndex) delete(ruleID string) {
	bucket := i.bucket(ruleID)
	bucket.Lock()
	delete(bucket.index, ruleID)
	bucket.Unlock()
}

func (i rulesIndex) get(ruleID string) (*Rule, error) {
	bucket := i.bucket(ruleID)
	bucket.RLock()
	defer bucket.RUnlock()
	rule, ok := bucket.index[ruleID]
	if !ok {
		return nil, fmt.Errorf("georule: rule %s not found", ruleID)
	}
	return rule, nil
}

type ruleSmallRegion struct {
	id      h3.H3Index
	mu      sync.RWMutex
	index   *rtree.RTree
	counter uint64
}

func newRuleSmallRegion(id h3.H3Index) *ruleSmallRegion {
	return &ruleSmallRegion{
		id:    id,
		index: &rtree.RTree{},
	}
}

func (r *ruleSmallRegion) isEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.counter == 0
}

func (r *ruleSmallRegion) delete(rule *Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.counter > 0 {
		r.counter--
	}
	r.index.Delete(
		[2]float64{rule.bbox.Min.X, rule.bbox.Min.Y},
		[2]float64{rule.bbox.Max.X, rule.bbox.Max.Y},
		rule,
	)
}

func (r *ruleSmallRegion) insertRule(rule *Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	bbox := rule.Bounding()
	r.counter++
	r.index.Insert(
		[2]float64{bbox.Min.X, bbox.Min.Y},
		[2]float64{bbox.Max.X, bbox.Max.Y},
		rule)
}

func (r *ruleSmallRegion) walk(ctx context.Context, device *Device, fn WalkRuleFunc) (err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.index.Search(
		[2]float64{device.Latitude, device.Longitude},
		[2]float64{device.Latitude, device.Longitude},
		func(_, _ [2]float64, value interface{}) bool {
			rule, ok := value.(*Rule)
			if ok {
				if err = fn(ctx, rule, nil); err != nil {
					return false
				}
			}
			return true
		},
	)
	return
}

type ruleLargeRegion struct {
	id    h3.H3Index
	mu    sync.RWMutex
	index map[string]*Rule
}

func newRuleLargeRegion(id h3.H3Index) *ruleLargeRegion {
	return &ruleLargeRegion{
		id:    id,
		index: make(map[string]*Rule),
	}
}

func (r *ruleLargeRegion) isEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.index) == 0
}

func (r *ruleLargeRegion) delete(rule *Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.index, rule.ruleID)
}

func (r *ruleLargeRegion) walk(ctx context.Context, _ *Device, fn WalkRuleFunc) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rule := range r.index {
		if err := fn(ctx, rule, nil); err != nil {
			return err
		}
	}
	return nil
}

func (r *ruleLargeRegion) insertRule(rule *Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.index[rule.ruleID] = rule
}

func (r *ruleLargeRegion) removeRule(ruleID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.index, ruleID)
}
