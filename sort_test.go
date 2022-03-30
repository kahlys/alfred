package alfred

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type elem struct {
	String string
	Int    int
	Float  float64
	Bool   bool
	Time   time.Time
}

func TestSort(t *testing.T) {
	bruce := elem{
		String: "Bruce",
		Int:    8,
		Float:  0.08,
		Bool:   true,
		Time:   time.Date(2006, time.May, 0, 0, 0, 0, 0, time.UTC),
	}
	diana := elem{
		String: "Diana",
		Int:    1,
		Float:  0.01,
		Bool:   false,
		Time:   time.Date(2026, time.May, 0, 0, 0, 0, 0, time.UTC),
	}
	clark := elem{
		String: "Clark",
		Int:    4,
		Float:  0.04,
		Bool:   true,
		Time:   time.Date(2016, time.May, 0, 0, 0, 0, 0, time.UTC),
	}
	tests := map[string]struct {
		opt  Option
		want []elem
	}{
		"text-asc":   {opt: Option{SortBy: "String", Order: orderASC}, want: []elem{bruce, clark, diana}},
		"text-desc":  {opt: Option{SortBy: "String", Order: orderDESC}, want: []elem{diana, clark, bruce}},
		"int-asc":    {opt: Option{SortBy: "Int", Order: orderASC}, want: []elem{diana, clark, bruce}},
		"int-desc":   {opt: Option{SortBy: "Int", Order: orderDESC}, want: []elem{bruce, clark, diana}},
		"float-asc":  {opt: Option{SortBy: "Float", Order: orderASC}, want: []elem{diana, clark, bruce}},
		"float-desc": {opt: Option{SortBy: "Float", Order: orderDESC}, want: []elem{bruce, clark, diana}},
		"bool-asc":   {opt: Option{SortBy: "Bool", Order: orderASC}, want: []elem{diana, bruce, clark}},
		"bool-desc":  {opt: Option{SortBy: "Bool", Order: orderDESC}, want: []elem{clark, bruce, diana}},
		"time-asc":   {opt: Option{SortBy: "Time", Order: orderASC}, want: []elem{bruce, clark, diana}},
		"time-desc":  {opt: Option{SortBy: "Time", Order: orderDESC}, want: []elem{diana, clark, bruce}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			arr := []elem{bruce, diana, clark}
			tt.opt.Sort(arr)
			assert.Equal(t, tt.want, arr)
		})
	}
}
