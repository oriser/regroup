package regroup

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

type Single struct {
	Duration time.Duration `regroup:"duration"`
}

type Including struct {
	Num int    `regroup:"num"`
	Str string `regroup:"str"`
	Single
}

type IncorrectGroup struct {
	Str string `regroup:"not_found"`
}

type Required struct {
	Str string `regroup:"str,required"`
}

type IncludingPointers struct {
	Num    *uint  `regroup:"num"`
	Str    string `regroup:"str"`
	Single *Single
}

type FloatBool struct {
	F float32 `regroup:"float"`
	B bool    `regroup:"bool"`
}

type NonExtracting struct {
	Single
	NonExtract string
}

func uintPtr(val uint) *uint {
	return &val
}

func getTarget(expected interface{}) interface{} {
	switch v := expected.(type) {
	case Single:
		return Single{}
	case *Single:
		return &Single{}
	case *Including:
		return &Including{}
	case *IncorrectGroup:
		return &IncorrectGroup{}
	case *Required:
		return &Required{}
	case *IncludingPointers:
		if v.Num != nil {
			return &IncludingPointers{Num: uintPtr(0), Single: &Single{}}
		}
		return &IncludingPointers{}
	case *FloatBool:
		return &FloatBool{}
	case *NonExtracting:
		return &NonExtracting{}
	default:
		panic("invalid expected")
	}
}

func isErrorMatch(t *testing.T, expected error, got error) {
	if got != nil {
		if expected == nil {
			t.Errorf("Unexpected error = %v", got)
		} else if reflect.ValueOf(expected).Elem().Type() != reflect.ValueOf(got).Elem().Type() {
			t.Errorf("Unexpected error type. Error = %T(%v), wantErr: %T(%v)", got, got, expected, expected)
		}
		if reflect.ValueOf(expected).Elem().Type() == reflect.ValueOf(fmt.Errorf("")).Elem().Type() {
			// if it's just an error string (*errors.errorString), check that the target error contains this error string
			if !strings.Contains(got.Error(), expected.Error()) {
				t.Errorf("Expected error to conatin the string: `%v`, but it's not: %v", expected, got)
			}
		}
		return
	}
	if expected != nil {
		t.Errorf("Expected error = %T(%v), got no error", expected, expected)
	}
}

func TestReGroup_MatchToTarget(t *testing.T) {
	r := MustCompile(`(?P<duration>.*?)\s+(?P<num>\d+)\s+(?P<str>.*)`)

	tests := []struct {
		name        string
		s           string
		wantErr     error
		expected    interface{}
		differentRe string
	}{
		{
			name:     "Single struct",
			s:        "5s 123 foo",
			expected: &Single{Duration: 5 * time.Second},
		},
		{
			name:     "Including struct",
			s:        "5s 123 foo",
			expected: &Including{Single: Single{Duration: 5 * time.Second}, Num: 123, Str: "foo"},
		},
		{
			name:        "Float bool",
			s:           "5.321 true",
			expected:    &FloatBool{F: 5.321, B: true},
			differentRe: `(?P<float>\d+\.\d+)\s+(?P<bool>.*)`,
		},
		{
			name:     "With no extract",
			s:        "5s 123 foo",
			expected: &NonExtracting{Single: Single{Duration: 5 * time.Second}, NonExtract: ""},
		},
		{
			name:        "Empty non required field",
			s:           "5s 123",
			expected:    &Including{Single: Single{Duration: 5 * time.Second}, Num: 123},
			differentRe: `(?P<duration>.*?)\s+(?P<num>\d+)\s*(?P<str>.*)?`,
		},
		{
			name:     "No match",
			s:        "5s aa foo",
			wantErr:  &NoMatchFoundError{},
			expected: &Single{},
		},
		{
			name:     "Invalid group name",
			s:        "5s 123 foo",
			wantErr:  &UnknownGroupError{},
			expected: &IncorrectGroup{},
		},
		{
			name:     "No struct pointer",
			s:        "5s 123 foo",
			wantErr:  &NotStructPtrError{},
			expected: Single{},
		},
		{
			name:     "Required",
			s:        "5s 123 foo",
			expected: &Required{Str: "foo"},
		},
		{
			name:     "Required no present",
			s:        "5s 123 ",
			wantErr:  &RequiredGroupIsEmpty{},
			expected: &Required{},
		},
		{
			name:     "Parse error duration",
			s:        "5ls 123 foo",
			wantErr:  &ParseError{},
			expected: &Single{},
		},
		{
			name:        "Parse error number",
			s:           "5s 123.3 foo",
			wantErr:     &ParseError{},
			expected:    &Including{},
			differentRe: `(?P<duration>.*?)\s+(?P<num>\d+\.\d+)\s+(?P<str>.*)`,
		},
		{
			name:        "Parse error bool",
			s:           "123.3 invalid_bool",
			wantErr:     &ParseError{},
			expected:    &FloatBool{},
			differentRe: `(?P<float>\d+\.\d+)\s+(?P<bool>.*)`,
		},
		{
			name:        "Parse error float",
			s:           "123.s3 true",
			wantErr:     &ParseError{},
			expected:    &FloatBool{},
			differentRe: `(?P<float>\d+\.s\d+)\s+(?P<bool>.*)`,
		},
		{
			name:        "Parse error uint",
			s:           "5s -3 str",
			wantErr:     &ParseError{},
			expected:    &IncludingPointers{Num: uintPtr(1)},
			differentRe: `(?P<duration>.*?)\s+(?P<num>-\d+)\s+(?P<str>.*)`,
		},
		{
			name:        "Compile error",
			s:           "",
			wantErr:     &CompileError{},
			expected:    &Single{},
			differentRe: "invlid[",
		},
		{
			name:     "Including struct pointer nil field",
			s:        "5s 123 foo",
			wantErr:  fmt.Errorf("can't set value to nil pointer in field"),
			expected: &IncludingPointers{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reGroup := r
			var err error
			if tt.differentRe != "" {
				reGroup, err = Compile(tt.differentRe)
				if err != nil {
					isErrorMatch(t, tt.wantErr, err)
					return
				}
			}
			target := getTarget(tt.expected)
			if err = reGroup.MatchToTarget(tt.s, target); err != nil || tt.wantErr != nil {
				isErrorMatch(t, tt.wantErr, err)
				return
			}
			if !reflect.DeepEqual(tt.expected, target) {
				t.Errorf("Expected: %+v, Got: %+v", tt.expected, target)
			}
		})
	}
}

func TestReGroup_MatchAllToTarget(t *testing.T) {
	r := MustCompile(`(?P<duration>.*?)\s+(?P<num>\d+)\s+(?P<str>.*)`)

	tests := []struct {
		name     string
		s        string
		wantErr  error
		expected []interface{}
	}{
		{
			name:     "Single match",
			s:        "5s 123 foo",
			expected: []interface{}{&Single{Duration: 5 * time.Second}},
		},
		{
			name:     "Including single match",
			s:        "5s 123 foo",
			expected: []interface{}{&Including{Single: Single{Duration: 5 * time.Second}, Num: 123, Str: "foo"}},
		},
		{
			name:     "No struct pointer",
			s:        "5s 123 foo",
			wantErr:  &NotStructPtrError{},
			expected: []interface{}{Single{}},
		},
		{
			name:     "No match",
			s:        "5s aa foo",
			wantErr:  &NoMatchFoundError{},
			expected: []interface{}{&Single{}},
		},
		{
			name:     "Parse error",
			s:        "5ls 123 foo",
			wantErr:  &ParseError{},
			expected: []interface{}{&Single{Duration: 5 * time.Second}},
		},
		{
			name:     "Multiple matches",
			s:        "5s 123 foo\n8h 123 foo",
			expected: []interface{}{&Single{Duration: 5 * time.Second}, &Single{Duration: 8 * time.Hour}},
		},
		{
			name:     "Including struct pointer",
			s:        "5s 123 foo",
			expected: []interface{}{&IncludingPointers{Single: &Single{Duration: 5 * time.Second}, Num: uintPtr(123), Str: "foo"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := getTarget(tt.expected[0])
			matches, err := r.MatchAllToTarget(tt.s, -1, target)
			if err != nil || tt.wantErr != nil {
				isErrorMatch(t, tt.wantErr, err)
				return
			}
			if len(matches) != len(tt.expected) {
				t.Errorf("Expected: %d matches, Got: %d", len(tt.expected), len(matches))
			}
			for i, match := range matches {
				if !reflect.DeepEqual(tt.expected[i], match) {
					t.Errorf("Not equal match at index %d. Expected: %+v, Got: %+v", i, tt.expected[i], match)
				}
			}
		})
	}
}
