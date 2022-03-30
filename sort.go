package alfred

import (
	"reflect"
	"sort"
	"strconv"
	"time"
	"unicode"
)

const (
	orderASC  = "asc"
	orderDESC = "desc"
)

// It panics if x is not a slice of structs.
func (flt Option) Sort(slice any) {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		panic("call of Option.Sort on " + v.Kind().String() + " value")
	}

	sort.Slice(slice, func(i, j int) bool {
		var res bool

		rvi, rvj := v.Index(i), v.Index(j)
		if rvi.Kind() != reflect.Struct {
			panic("call of Option.Sort on a slice of" + rvi.Kind().String() + " value")
		}

		xi, xj := -1, -1
		for x := 0; x < rvi.NumField(); x++ {
			ti, tj := rvi.Type().Field(x), rvj.Type().Field(x)
			// optimize and research for tag also
			if ti.Name == flt.SortBy {
				xi = x
			}
			if tj.Name == flt.SortBy {
				xj = x
			}
		}

		if xi < 0 || xj < 0 {
			return false
		}

		vi, vj := rvi.Field(xi), rvj.Field(xj)

		if vi.Kind() != vj.Kind() {
			panic("call of Option.Sort with a slices of different structs")
		}

		switch vi.Interface().(type) {
		case string:
			res = stringsLess(vi.String(), vj.String())
		case int, int8, int16, int32, int64:
			res = vi.Int() < vj.Int()
		case float32, float64:
			res = vi.Float() < vj.Float()
		case bool:
			res = strconv.FormatBool(vi.Bool()) < strconv.FormatBool(vj.Bool())
		case time.Time:
			viTime, ok := vi.Interface().(time.Time)
			if !ok {
				panic("NOK")
			}
			vjTime, ok := vj.Interface().(time.Time)
			if !ok {
				panic("NOK")
			}
			res = viTime.Before(vjTime)
		default:
			return false
		}

		return res != (flt.Order == orderDESC)
	})
}

// alphabetical order
func stringsLess(a, b string) bool {
	if a == "" {
		return true
	} else if b == "" {
		return false
	}

	iRunes := []rune(a)
	jRunes := []rune(b)

	var min = len(iRunes)
	if min > len(jRunes) {
		min = len(jRunes)
	}

	for idx := 0; idx < min; idx++ {
		var ir = iRunes[idx]
		var jr = jRunes[idx]

		var lir = unicode.ToLower(ir)
		var ljr = unicode.ToLower(jr)

		if lir != ljr {
			return lir < ljr
		}

		// If the lowercase runes are the same, compare the original
		if ir != jr {
			return ir < jr
		}
	}
	return len(a) < len(b)
}
