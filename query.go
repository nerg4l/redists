package redists

import (
	"context"
	"strconv"
	"time"
)

type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

func parseDataPoint(is []interface{}) DataPoint {
	v, _ := strconv.ParseFloat(parseString(is[1]), 64)
	return DataPoint{Timestamp: time.UnixMilli(is[0].(int64)), Value: v}
}

const (
	nameRange    = nameRanger("TS.RANGE")
	nameRevRange = nameRanger("TS.REVRANGE")
)

type nameRanger string

type CmdRanger struct {
	name        nameRanger
	key         string
	from        Timestamp
	to          Timestamp
	tsFilter    []time.Time
	valueFilter *ValueFilter
	count       *int64
	align       *Timestamp
	aggregation *Aggregation
}

func newCmdRanger(name nameRanger, key string, from Timestamp, to Timestamp) *CmdRanger {
	return &CmdRanger{name: name, key: key, from: from, to: to}
}

func (c *CmdRanger) Name() string {
	return string(c.name)
}

func (c *CmdRanger) Args() []interface{} {
	args := []interface{}{c.key, c.from.Arg(), c.to.Arg()}
	if len(c.tsFilter) > 0 {
		args = append(args, optionNameFilterByTS)
		for i := range c.tsFilter {
			args = append(args, c.tsFilter[i].UnixMilli())
		}
	}
	if c.valueFilter != nil {
		args = append(args, optionNameFilterByValue, c.valueFilter.Min, c.valueFilter.Max)
	}
	if c.count != nil {
		args = append(args, optionNameCount, *c.count)
	}
	if c.align != nil {
		args = append(args, optionNameAlign, c.align.Arg())
	}
	if c.aggregation != nil {
		args = append(args, optionNameAggregation, string(c.aggregation.Type), c.aggregation.TimeBucket.Milliseconds())
	}
	return args
}

type OptionRanger func(cmd *CmdRanger)

// Range queries a range in forward direction.
func (c *Client) Range(ctx context.Context, key string, from Timestamp, to Timestamp, options ...OptionRanger) ([]DataPoint, error) {
	return c.ranger(ctx, nameRange, key, from, to, options...)
}

// RevRange queries a range in reverse direction.
func (c *Client) RevRange(ctx context.Context, key string, from Timestamp, to Timestamp, options ...OptionRanger) ([]DataPoint, error) {
	return c.ranger(ctx, nameRevRange, key, from, to, options...)
}

func (c *Client) ranger(ctx context.Context, name nameRanger, key string, from Timestamp, to Timestamp, options ...OptionRanger) ([]DataPoint, error) {
	cmd := newCmdRanger(name, key, from, to)
	for i := range options {
		options[i](cmd)
	}
	res, err := c.d.Do(ctx, cmd.Name(), cmd.Args()...)
	var ds []DataPoint
	if is, ok := res.([]interface{}); ok {
		ds = make([]DataPoint, len(is))
		for i := range is {
			ds[i] = parseDataPoint(is[i].([]interface{}))
		}
	}
	return ds, err
}

func RangerWithTSFilter(tss ...time.Time) OptionRanger {
	return func(cmd *CmdRanger) {
		cmd.tsFilter = tss
	}
}

func RangerWithValueFilter(min float64, max float64) OptionRanger {
	return func(cmd *CmdRanger) {
		cmd.valueFilter = &ValueFilter{Min: min, Max: max}
	}
}

func RangerWithCount(c int64) OptionRanger {
	return func(cmd *CmdRanger) {
		cmd.count = &c
	}
}

func RangerWithAlign(a Timestamp) OptionRanger {
	return func(cmd *CmdRanger) {
		cmd.align = &a
	}
}

func RangerWithAggregation(t AggregationType, timeBucket time.Duration) OptionRanger {
	return func(cmd *CmdRanger) {
		cmd.aggregation = &Aggregation{Type: t, TimeBucket: timeBucket}
	}
}

type TimeSeries struct {
	Key        string
	Labels     []Label
	DataPoints []DataPoint
}

func parseTimeSeries(is []interface{}) TimeSeries {
	var ls []Label
	if ils, ok := is[1].([]interface{}); ok {
		ls = make([]Label, len(ils))
		for i := range ils {
			ls[i] = parseLabel(ils[i].([]interface{}))
		}
	}
	var dps []DataPoint
	if idps, ok := is[2].([]interface{}); ok {
		dps = make([]DataPoint, len(idps))
		for i := range idps {
			dps[i] = parseDataPoint(idps[i].([]interface{}))
		}
	}
	return TimeSeries{
		Key:        parseString(is[0]),
		Labels:     ls,
		DataPoints: dps,
	}
}

const (
	nameMRange    = nameMRanger("TS.MRANGE")
	nameMRevRange = nameMRanger("TS.MREVRANGE")
)

type nameMRanger string

type CmdMRanger struct {
	name        nameMRanger
	from        Timestamp
	to          Timestamp
	filters     []Filter
	tsFilter    []time.Time
	valueFilter *ValueFilter
	withLabels  []string
	count       *int64
	align       *Timestamp
	aggregation *Aggregation
	groupBy     *GroupBy
}

func newCmdMRanger(name nameMRanger, from Timestamp, to Timestamp, filters []Filter) *CmdMRanger {
	return &CmdMRanger{name: name, from: from, to: to, filters: filters}
}

func (c *CmdMRanger) Name() string {
	return string(c.name)
}

func (c *CmdMRanger) Args() []interface{} {
	args := []interface{}{c.from.Arg(), c.to.Arg()}
	if len(c.tsFilter) > 0 {
		args = append(args, optionNameFilterByTS)
		for i := range c.tsFilter {
			args = append(args, c.tsFilter[i].UnixMilli())
		}
	}
	if c.valueFilter != nil {
		args = append(args, optionNameFilterByValue, c.valueFilter.Min, c.valueFilter.Max)
	}
	if c.withLabels != nil {
		if len(c.withLabels) == 0 {
			args = append(args, optionNameWithLabels)
		} else {
			args = append(args, optionNameSelectedLabels)
			for i := range c.withLabels {
				args = append(args, c.withLabels[i])
			}
		}
	}
	if c.count != nil {
		args = append(args, optionNameCount, *c.count)
	}
	if c.align != nil {
		args = append(args, optionNameAlign, c.align.Arg())
	}
	if c.aggregation != nil {
		args = append(args, optionNameAggregation, string(c.aggregation.Type), c.aggregation.TimeBucket.Milliseconds())
	}
	args = append(args, optionNameFilter)
	for i := range c.filters {
		args = append(args, c.filters[i].Arg())
	}
	if c.groupBy != nil {
		args = append(args, optionNameGroupBy, c.groupBy.Label, optionNameReduce, string(c.groupBy.Reducer))
	}
	return args
}

type OptionMRanger func(cmd *CmdMRanger)

// MRange queries a range across multiple time-series by filters in forward direction.
func (c *Client) MRange(ctx context.Context, from Timestamp, to Timestamp, filters []Filter, options ...OptionMRanger) ([]TimeSeries, error) {
	return c.mRanger(ctx, nameMRange, from, to, filters, options...)
}

// MRevRange queries a range across multiple time-series by filters in reverse direction.
func (c *Client) MRevRange(ctx context.Context, from Timestamp, to Timestamp, filters []Filter, options ...OptionMRanger) ([]TimeSeries, error) {
	return c.mRanger(ctx, nameMRevRange, from, to, filters, options...)
}

func (c *Client) mRanger(ctx context.Context, name nameMRanger, from Timestamp, to Timestamp, filters []Filter, options ...OptionMRanger) ([]TimeSeries, error) {
	cmd := newCmdMRanger(name, from, to, filters)
	for i := range options {
		options[i](cmd)
	}
	res, err := c.d.Do(ctx, cmd.Name(), cmd.Args()...)
	var ds []TimeSeries
	if is, ok := res.([]interface{}); ok {
		ds = make([]TimeSeries, len(is))
		for i := range is {
			ds[i] = parseTimeSeries(is[i].([]interface{}))
		}
	}
	return ds, err
}

func MRangerWithTSFilter(tss ...time.Time) OptionMRanger {
	return func(cmd *CmdMRanger) {
		cmd.tsFilter = tss
	}
}

func MRangerWithValueFilter(min float64, max float64) OptionMRanger {
	return func(cmd *CmdMRanger) {
		cmd.valueFilter = &ValueFilter{Min: min, Max: max}
	}
}

func MRangerWithLabels(labels ...string) OptionMRanger {
	return func(cmd *CmdMRanger) {
		if labels == nil {
			cmd.withLabels = []string{}
		} else {
			cmd.withLabels = labels
		}
	}
}

func MRangerWithCount(c int64) OptionMRanger {
	return func(cmd *CmdMRanger) {
		cmd.count = &c
	}
}

func MRangerWithAlign(a Timestamp) OptionMRanger {
	return func(cmd *CmdMRanger) {
		cmd.align = &a
	}
}

func MRangerWithAggregation(t AggregationType, timeBucket time.Duration) OptionMRanger {
	return func(cmd *CmdMRanger) {
		cmd.aggregation = &Aggregation{Type: t, TimeBucket: timeBucket}
	}
}

func MRangerWithGroupBy(label string, reducer ReducerType) OptionMRanger {
	return func(cmd *CmdMRanger) {
		cmd.groupBy = &GroupBy{Label: label, Reducer: reducer}
	}
}

type LastDatapoint struct {
	Key       string
	Labels    []Label
	DataPoint DataPoint
}

func parseLastDatapoint(is []interface{}) LastDatapoint {
	var ls []Label
	if ils, ok := is[1].([]interface{}); ok {
		ls = make([]Label, len(ils))
		for i := range ils {
			ls[i] = parseLabel(ils[i].([]interface{}))
		}
	}
	return LastDatapoint{
		Key:       parseString(is[0]),
		Labels:    ls,
		DataPoint: parseDataPoint(is[2].([]interface{})),
	}
}

type CmdGet struct {
	key string
}

func newCmdGet(key string) *CmdGet {
	return &CmdGet{key: key}
}

func (c *CmdGet) Name() string {
	return "TS.GET"
}

func (c *CmdGet) Args() []interface{} {
	return []interface{}{c.key}
}

// Get gets the last sample.
func (c *Client) Get(ctx context.Context, key string) (DataPoint, error) {
	cmd := newCmdGet(key)
	res, err := c.d.Do(ctx, cmd.Name(), cmd.Args()...)
	if err != nil {
		return DataPoint{}, err
	}
	return parseDataPoint(res.([]interface{})), nil
}

type CmdMGet struct {
	filters    []Filter
	withLabels []string
}

func newCmdMGet(filters []Filter) *CmdMGet {
	return &CmdMGet{filters: filters}
}

func (c *CmdMGet) Name() string {
	return "TS.MGET"
}

func (c *CmdMGet) Args() []interface{} {
	var args []interface{}
	if c.withLabels != nil {
		if len(c.withLabels) == 0 {
			args = append(args, optionNameWithLabels)
		} else {
			args = append(args, optionNameSelectedLabels)
			for i := range c.withLabels {
				args = append(args, c.withLabels[i])
			}
		}
	}
	args = append(args, optionNameFilter)
	for i := range c.filters {
		args = append(args, c.filters[i].Arg())
	}
	return args
}

type OptionMGet func(cmd *CmdMGet)

// MGet gets the last samples matching the specific filter.
func (c *Client) MGet(ctx context.Context, filters []Filter, options ...OptionMGet) ([]LastDatapoint, error) {
	cmd := newCmdMGet(filters)
	for i := range options {
		options[i](cmd)
	}
	res, err := c.d.Do(ctx, cmd.Name(), cmd.Args()...)
	var ds []LastDatapoint
	if is, ok := res.([]interface{}); ok {
		ds = make([]LastDatapoint, len(is))
		for i := range is {
			ds[i] = parseLastDatapoint(is[i].([]interface{}))
		}
	}
	return ds, err
}

func MGetWithLabels(labels ...string) OptionMGet {
	return func(cmd *CmdMGet) {
		if labels == nil {
			cmd.withLabels = []string{}
		} else {
			cmd.withLabels = labels
		}
	}
}