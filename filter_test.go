package alfred

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFilterFromURLValues(t *testing.T) {
	tests := map[string]struct {
		url  string
		want Filter
	}{
		"like":    {url: "?filter[name][like]=batman", want: Like{"name", "batman"}},
		"eq":      {url: "?filter[name][eq]=batman", want: EQ{"name", "batman"}},
		"gt":      {url: "?filter[age][gt]=42", want: GT{"age", "42"}},
		"gte":     {url: "?filter[age][gte]=42", want: GTE{"age", "42"}},
		"lt":      {url: "?filter[age][lt]=42", want: LT{"age", "42"}},
		"lte":     {url: "?filter[age][lte]=42", want: LTE{"age", "42"}},
		"contain": {url: "?filter[age][contain]=42", want: Contain{"age", "42"}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			url, err := url.Parse(tt.url)
			assert.NoError(t, err)
			f := ParseURLValues(url.Query())
			assert.Equal(t, tt.want, f.Filters[0])
		})
	}
}

func TestToSQL(t *testing.T) {
	tests := map[string]struct {
		filter Filter
		want   string
	}{
		"like": {filter: Like{"name", "batman"}, want: `"name" ILIKE '%batman%'`},
		"eq":   {filter: EQ{"name", "batman"}, want: `"name" = 'batman'`},
		"gt":   {filter: GT{"age", "42"}, want: `"age" > '42'`},
		"gte":  {filter: GTE{"age", "42"}, want: `"age" >= '42'`},
		"lt":   {filter: LT{"age", "42"}, want: `"age" < '42'`},
		"lte":  {filter: LTE{"age", "42"}, want: `"age" <= '42'`},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			actual := tt.filter.ToSQL()
			assert.Equal(t, tt.want, actual)
		})
	}
}

func TestKeep(t *testing.T) {
	data := struct {
		Name  string `filter:"name"`
		Alive bool   `filter:"alive"`
		Age   int    `filter:"age"`
	}{
		Name:  "Bruce,Wayne",
		Alive: true,
		Age:   39,
	}
	tests := map[string]struct {
		filter Filter
		want   bool
	}{
		"like-keep":     {filter: Like{"name", "bruce"}, want: true},
		"like-drop":     {filter: Like{"name", "diana"}, want: false},
		"eq-keep":       {filter: EQ{"alive", "1"}, want: true},
		"eq-drop":       {filter: EQ{"alive", "0"}, want: false},
		"gt-keep":       {filter: GT{"age", "32"}, want: true},
		"gt-drop":       {filter: GT{"age", "39"}, want: false},
		"gte-keep":      {filter: GTE{"age", "39"}, want: true},
		"gte-drop":      {filter: GTE{"age", "42"}, want: false},
		"lt-keep":       {filter: LT{"age", "42"}, want: true},
		"lt-drop":       {filter: LT{"age", "39"}, want: false},
		"lte-keep":      {filter: LTE{"age", "39"}, want: true},
		"lte-drop":      {filter: LTE{"age", "24"}, want: false},
		"contains-keep": {filter: Contain{"name", "Bruce"}, want: true},
		"contains-drop": {filter: Contain{"name", "Diana"}, want: false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			actual, err := tt.filter.Keep(data)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, actual)
		})
	}
}

func TestParseSortAndPageFromURLValues(t *testing.T) {
	endpoint := "?offset=1&limit=2&orderBy=ASC&sortBy=name"
	want := Option{Offset: 1, Limit: 2, SortBy: "name", Order: "ASC"}
	url, err := url.Parse(endpoint)
	assert.NoError(t, err)
	f := ParseURLValues(url.Query())
	assert.Equal(t, want, f)
}
