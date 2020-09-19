package regroup

import (
	"reflect"
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

func uintPtr(val uint) *uint {
	return &val
}

func getTarget(expected interface{}) interface{} {
	switch expected.(type) {
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
		return &IncludingPointers{Num: uintPtr(0), Single: &Single{}}
	case *FloatBool:
		return &FloatBool{}
	default:
		panic("invalid expected")
	}
}

func isErrorMatch(t *testing.T, expected error, got error) {
	if got != nil {
		if expected == nil {
			t.Errorf("MatchToTarget() unexpected error = %v", got)
		} else if reflect.ValueOf(expected).Elem().Type() != reflect.ValueOf(got).Elem().Type() {
			t.Errorf("MatchToTarget() unexpected error type. Error = %T(%v), wantErr: %T(%v)", got, got, expected, expected)
		}
		return
	}
	if expected != nil {
		t.Errorf("MatchToTarget() expected error = %T(%v), got no error", expected, expected)
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
			name:     "Parse error",
			s:        "5ls 123 foo",
			wantErr:  &ParseError{},
			expected: &Single{},
		},
		{
			name:        "Compile error",
			s:           "",
			wantErr:     &CompileError{},
			expected:    &Single{},
			differentRe: "invlid[",
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
			name:     "No match",
			s:        "5s aa foo",
			wantErr:  &NoMatchFoundError{},
			expected: []interface{}{&Single{}},
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
