package alfred

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Option are filtering, sorting and pagination parameters.
type Option struct {
	// pagination
	Limit  int
	Offset int

	// sorting
	Order  string
	SortBy string

	// filters
	Filters Filters
}

var paramFilter = regexp.MustCompile(`([a-zA-Z1-9_\-]+)\[([a-zA-Z1-9_\-]+)\]\[([a-zA-Z1-9_\-]+)\]`)

// ParseURLValues parse filters from url values. The format is '?filter[<>][<type>]=<value>'.
// Possible type are : like, eq, gt, gte, lt, lte.
func ParseURLValues(values url.Values) Option {
	var err error
	f := Option{}

	f.Limit, err = strconv.Atoi(values.Get("limit"))
	if err != nil {
		f.Limit = 0
	}
	offsetS := values.Get("offset")
	f.Offset, err = strconv.Atoi(offsetS)
	if err != nil {
		f.Offset = 0
	}

	f.SortBy = values.Get("sortBy")
	f.Order = values.Get("orderBy")

	for k, v := range values {
		matches := paramFilter.FindStringSubmatch(k)
		if len(matches) != 4 || matches[1] != "filter" {
			continue
		}

		switch matches[3] {
		case "like":
			f.Filters = append(f.Filters, Like{matches[2], v[0]})
		case "eq":
			f.Filters = append(f.Filters, EQ{matches[2], v[0]})
		case "gt":
			f.Filters = append(f.Filters, GT{matches[2], v[0]})
		case "gte":
			f.Filters = append(f.Filters, GTE{matches[2], v[0]})
		case "lt":
			f.Filters = append(f.Filters, LT{matches[2], v[0]})
		case "lte":
			f.Filters = append(f.Filters, LTE{matches[2], v[0]})
		case "contain":
			f.Filters = append(f.Filters, Contain{matches[2], v[0]})
		default:
			continue
		}
	}

	return f
}

// Filters for filtering result values.
type Filters []Filter

// Keep return true if the struct v has valid fields value according to applayable filters.
func (fs Filters) Keep(v any) (bool, error) {
	for _, f := range fs {
		keep, err := f.Keep(v)
		if !keep {
			return keep, err
		}
	}
	return true, nil
}

// AddToPSQLQuery add filter to a pq query.
// SELECT *, count(*) OVER() FROM (<query>) AS query WHERE <filter1> ILIKE <value1> AND ... ORDER BY <sortBy> <order> LIMIT 1 OFFSET 1
func AddToPSQLQuery(query string, opt Option) string {
	filterStr := ""
	where := make([]string, len(opt.Filters))
	i := 0
	for _, fv := range opt.Filters {
		where[i] = fv.ToSQL()
		i++
	}
	if len(where) > 0 {
		filterStr = fmt.Sprintf("WHERE %s", strings.Join(where, " AND "))
	}
	query = fmt.Sprintf("SELECT *, count(*) OVER() FROM (%v) AS query %v", query, filterStr)

	// sorting
	if opt.SortBy != "" {
		query = fmt.Sprintf("%v ORDER BY %v %v", query, pq.QuoteIdentifier(opt.SortBy), opt.Order)
	}
	// pagination
	limit := fmt.Sprintf("%v", opt.Limit)
	if opt.Limit <= 0 {
		limit = "ALL"
	}
	offset := fmt.Sprintf("%v", opt.Offset)
	if opt.Offset <= 0 {
		offset = "0"
	}
	return fmt.Sprintf("%v LIMIT %v OFFSET %v", query, limit, offset)
}

// Filter is the interface for a simple filter for filtering result values.
type Filter interface {
	Keep(v any) (bool, error)
	ToSQL() string
}

// Like filter to test if a string contains a substring.
type Like struct {
	Param string
	Value string
}

// Keep return true if the struct v has valid fields value according to the filter.
func (f Like) Keep(v any) (bool, error) {
	rv := reflect.ValueOf(v)
	t := rv.Type()
	if rv.Kind() != reflect.Struct {
		return false, fmt.Errorf("filter : not a struct (%v)", reflect.TypeOf(v).String())
	}
	for i := 0; i < rv.NumField(); i++ {
		ti := t.Field(i)
		if f.Param != ti.Name && f.Param != ti.Tag.Get("filter") {
			continue
		}
		if !stringsContainsI(rv.Field(i).String(), f.Value) {
			return false, nil
		}
	}
	return true, nil
}

func stringsContainsI(s string, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ToSQL return a SQL condition to be used in a SQL query : "Param ILIKE %Value%".
func (f Like) ToSQL() string {
	return fmt.Sprintf("%s ILIKE %s", pq.QuoteIdentifier(f.Param), pq.QuoteLiteral("%"+f.Value+"%"))
}

// EQ for ==
type EQ struct {
	Param string
	Value string
}

// Keep return true if the struct v has valid fields value according to the filter.
func (f EQ) Keep(v any) (bool, error) {
	rv := reflect.ValueOf(v)
	t := rv.Type()
	if rv.Kind() != reflect.Struct {
		return false, fmt.Errorf("filter : not a struct (%v)", reflect.TypeOf(v).String())
	}
	for i := 0; i < rv.NumField(); i++ {
		ti := t.Field(i)
		if f.Param != ti.Name && f.Param != ti.Tag.Get("filter") {
			continue
		}
		switch cv := rv.Field(i).Interface().(type) {
		case string:
			if f.Value != rv.Field(i).String() {
				return false, nil
			}
		case int, int8, int16, int32, int64:
			value, err := strconv.ParseInt(f.Value, 10, 0)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value != rv.Field(i).Int() {
				return false, nil
			}
		case float32, float64:
			value, err := strconv.ParseFloat(f.Value, 64)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value != rv.Field(i).Float() {
				return false, nil
			}
		case bool:
			value, err := strconv.ParseBool(f.Value)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value != rv.Field(i).Bool() {
				return false, nil
			}
		case time.Time:
			value, err := time.Parse(time.RFC3339, f.Value)
			if err != nil {
				return false, ErrParamType{f.Param, fmt.Errorf("%w : supported format time.RFC3339", err)}
			}
			if !value.Equal(cv) {
				return false, nil
			}
		default:
			return false, fmt.Errorf("filter : unsuported field type (%v as %v)", ti.Name, ti.Type.String())
		}
	}
	return true, nil
}

// ToSQL return a SQL condition to be used in a SQL query : "Param = Value".
func (f EQ) ToSQL() string {
	return fmt.Sprintf("%s = %s", pq.QuoteIdentifier(f.Param), pq.QuoteLiteral(f.Value))
}

// GT for >
type GT struct {
	Param string
	Value string
}

// Keep return true if the struct v has valid fields value according to the filter.
func (f GT) Keep(v any) (bool, error) {
	rv := reflect.ValueOf(v)
	t := rv.Type()
	if rv.Kind() != reflect.Struct {
		return false, fmt.Errorf("filter : not a struct (%v)", reflect.TypeOf(v).String())
	}
	for i := 0; i < rv.NumField(); i++ {
		ti := t.Field(i)
		if f.Param != ti.Name && f.Param != ti.Tag.Get("filter") {
			continue
		}
		switch cv := rv.Field(i).Interface().(type) {
		case int, int8, int16, int32, int64:
			value, err := strconv.ParseInt(f.Value, 10, 0)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value >= rv.Field(i).Int() {
				return false, nil
			}
		case float32, float64:
			value, err := strconv.ParseFloat(f.Value, 64)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value >= rv.Field(i).Float() {
				return false, nil
			}
		case time.Time:
			value, err := time.Parse(time.RFC3339, f.Value)
			if err != nil {
				return false, ErrParamType{f.Param, fmt.Errorf("%w : supported format time.RFC3339", err)}
			}
			if value.After(cv) || value.Equal(cv) {
				return false, nil
			}
		default:
			return false, fmt.Errorf("filter : unsuported field type (%v as %v)", ti.Name, ti.Type.String())
		}
	}
	return true, nil
}

// ToSQL return a SQL condition to be used in a SQL query : "Param > Value".
func (f GT) ToSQL() string {
	return fmt.Sprintf("%s > %s", pq.QuoteIdentifier(f.Param), pq.QuoteLiteral(f.Value))
}

// GTE for >=
type GTE struct {
	Param string
	Value string
}

// Keep return true if the struct v has valid fields value according to the filter.
func (f GTE) Keep(v any) (bool, error) {
	rv := reflect.ValueOf(v)
	t := rv.Type()
	if rv.Kind() != reflect.Struct {
		return false, fmt.Errorf("filter : not a struct (%v)", reflect.TypeOf(v).String())
	}
	for i := 0; i < rv.NumField(); i++ {
		ti := t.Field(i)
		if f.Param != ti.Name && f.Param != ti.Tag.Get("filter") {
			continue
		}
		switch cv := rv.Field(i).Interface().(type) {
		case int, int8, int16, int32, int64:
			value, err := strconv.ParseInt(f.Value, 10, 0)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value > rv.Field(i).Int() {
				return false, nil
			}
		case float32, float64:
			value, err := strconv.ParseFloat(f.Value, 64)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value > rv.Field(i).Float() {
				return false, nil
			}
		case time.Time:
			value, err := time.Parse(time.RFC3339, f.Value)
			if err != nil {
				return false, ErrParamType{f.Param, fmt.Errorf("%w : supported format time.RFC3339", err)}
			}
			if value.After(cv) {
				return false, nil
			}
		default:
			return false, fmt.Errorf("filter : unsuported field type (%v as %v)", ti.Name, ti.Type.String())
		}
	}
	return true, nil
}

// ToSQL return a SQL condition to be used in a SQL query : "Param >= Value".
func (f GTE) ToSQL() string {
	return fmt.Sprintf("%s >= %s", pq.QuoteIdentifier(f.Param), pq.QuoteLiteral(f.Value))
}

// LT for <
type LT struct {
	Param string
	Value string
}

// Keep return true if the struct v has valid fields value according to the filter.
func (f LT) Keep(v any) (bool, error) {
	rv := reflect.ValueOf(v)
	t := rv.Type()
	if rv.Kind() != reflect.Struct {
		return false, fmt.Errorf("filter : not a struct (%v)", reflect.TypeOf(v).String())
	}
	for i := 0; i < rv.NumField(); i++ {
		ti := t.Field(i)
		if f.Param != ti.Name && f.Param != ti.Tag.Get("filter") {
			continue
		}
		switch cv := rv.Field(i).Interface().(type) {
		case int, int8, int16, int32, int64:
			value, err := strconv.ParseInt(f.Value, 10, 0)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value <= rv.Field(i).Int() {
				return false, nil
			}
		case float32, float64:
			value, err := strconv.ParseFloat(f.Value, 64)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value <= rv.Field(i).Float() {
				return false, nil
			}
		case time.Time:
			value, err := time.Parse(time.RFC3339, f.Value)
			if err != nil {
				return false, ErrParamType{f.Param, fmt.Errorf("%w : supported format time.RFC3339", err)}
			}
			if value.Before(cv) || value.Equal(cv) {
				return false, nil
			}
		default:
			return false, fmt.Errorf("filter : unsuported field type (%v as %v)", ti.Name, ti.Type.String())
		}
	}
	return true, nil
}

// ToSQL return a SQL condition to be used in a SQL query : "Param < Value".
func (f LT) ToSQL() string {
	return fmt.Sprintf("%s < %s", pq.QuoteIdentifier(f.Param), pq.QuoteLiteral(f.Value))
}

// LTE for <=
type LTE struct {
	Param string
	Value string
}

// Keep return true if the struct v has valid fields value according to the filter.
func (f LTE) Keep(v any) (bool, error) {
	rv := reflect.ValueOf(v)
	t := rv.Type()
	if rv.Kind() != reflect.Struct {
		return false, fmt.Errorf("filter : not a struct (%v)", reflect.TypeOf(v).String())
	}
	for i := 0; i < rv.NumField(); i++ {
		ti := t.Field(i)
		if f.Param != ti.Name && f.Param != ti.Tag.Get("filter") {
			continue
		}
		switch cv := rv.Field(i).Interface().(type) {
		case int, int8, int16, int32, int64:
			value, err := strconv.ParseInt(f.Value, 10, 0)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value < rv.Field(i).Int() {
				return false, nil
			}
		case float32, float64:
			value, err := strconv.ParseFloat(f.Value, 64)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value < rv.Field(i).Float() {
				return false, nil
			}
		case time.Time:
			value, err := time.Parse(time.RFC3339, f.Value)
			if err != nil {
				return false, ErrParamType{f.Param, fmt.Errorf("%w : supported format time.RFC3339", err)}
			}
			if value.Before(cv) {
				return false, nil
			}
		default:
			return false, fmt.Errorf("filter : unsuported field type (%v as %v)", ti.Name, ti.Type.String())
		}
	}
	return true, nil
}

// ToSQL return a SQL condition to be used in a SQL query : "Param <= Value".
func (f LTE) ToSQL() string {
	return fmt.Sprintf("%s <= %s", pq.QuoteIdentifier(f.Param), pq.QuoteLiteral(f.Value))
}

type Contain struct {
	Param string
	Value string
}

// Keep return true if the struct v has valid fields value according to the filter.
func (f Contain) Keep(v any) (bool, error) {
	rv := reflect.ValueOf(v)
	t := rv.Type()
	if rv.Kind() != reflect.Struct {
		return false, fmt.Errorf("filter : not a struct (%v)", reflect.TypeOf(v).String())
	}
	for i := 0; i < rv.NumField(); i++ {
		ti := t.Field(i)
		if f.Param != ti.Name && f.Param != ti.Tag.Get("filter") {
			continue
		}
		switch rv.Field(i).Interface().(type) {
		// kv value is an array with , as separator (x,x,x,x,...)
		case string:
			values := strings.Split(rv.Field(i).String(), ",")
			for _, elem := range values {
				if f.Value == elem {
					return true, nil
				}
			}
			return false, nil
		case int, int8, int16, int32, int64:
			value, err := strconv.ParseInt(f.Value, 10, 0)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value != rv.Field(i).Int() {
				return false, nil
			}
		case float32, float64:
			value, err := strconv.ParseFloat(f.Value, 64)
			if err != nil {
				return false, ErrParamType{f.Param, err}
			}
			if value != rv.Field(i).Float() {
				return false, nil
			}
		default:
			return false, fmt.Errorf("filter : unsuported field type (%v as %v)", ti.Name, ti.Type.String())
		}
	}
	return true, nil
}

// ToSQL return a SQL condition to be used in a SQL query : "Param <= Value".
func (f Contain) ToSQL() string {
	return fmt.Sprintf("string_to_array(%s, ',') @> %v", pq.QuoteIdentifier(f.Param), pq.QuoteLiteral("{"+f.Value+"}"))
}

// An ErrParamType is returned when failed to convert to a certain type a value of a filter.
type ErrParamType struct {
	Param string
	Err   error
}

func (e ErrParamType) Error() string {
	return fmt.Sprintf("filter: field '%v' : %v", e.Param, e.Err)
}
