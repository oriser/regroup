package regroup

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type NonExtracting struct {
	Single
	NonExtract string
}

func uintPtr(val uint) *uint {
	return &val
}

func getTarget(t *testing.T, expected interface{}) interface{} {
	t.Helper()

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
	t.Helper()

	if expected == nil {
		require.NoError(t, got)
	}
	require.Error(t, got)
	require.IsType(t, expected, got)

	if reflect.ValueOf(expected).Elem().Type() == reflect.ValueOf(fmt.Errorf("")).Elem().Type() {
		// if it's just an error string (*errors.errorString), check that the target error contains this error string
		require.Contains(t, got.Error(), expected.Error())
	}

}

func TestMatchToTarget(t *testing.T) {
	r := MustCompile(`(?P<duration>.*?)\s+(?P<num>\d+)\s+(?P<str>.*)`)

	type differentRe struct {
		re          string
		mustCompile bool
		shouldPanic bool
	}

	tests := []struct {
		name     string
		s        string
		wantErr  error
		expected interface{}
		differentRe
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
			differentRe: differentRe{re: `(?P<float>\d+\.\d+)\s+(?P<bool>.*)`},
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
			differentRe: differentRe{re: `(?P<duration>.*?)\s+(?P<num>\d+)\s*(?P<str>.*)?`},
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
			differentRe: differentRe{re: `(?P<duration>.*?)\s+(?P<num>\d+\.\d+)\s+(?P<str>.*)`},
		},
		{
			name:        "Parse error bool",
			s:           "123.3 invalid_bool",
			wantErr:     &ParseError{},
			expected:    &FloatBool{},
			differentRe: differentRe{re: `(?P<float>\d+\.\d+)\s+(?P<bool>.*)`},
		},
		{
			name:        "Parse error float",
			s:           "123.s3 true",
			wantErr:     &ParseError{},
			expected:    &FloatBool{},
			differentRe: differentRe{re: `(?P<float>\d+\.s\d+)\s+(?P<bool>.*)`},
		},
		{
			name:        "Parse error uint",
			s:           "5s -3 str",
			wantErr:     &ParseError{},
			expected:    &IncludingPointers{Num: uintPtr(1)},
			differentRe: differentRe{re: `(?P<duration>.*?)\s+(?P<num>-\d+)\s+(?P<str>.*)`},
		},
		{
			name:        "Compile error",
			s:           "",
			wantErr:     &CompileError{},
			expected:    &Single{},
			differentRe: differentRe{re: "invlid["},
		},
		{
			name:        "Compile error panic",
			s:           "",
			expected:    &Single{},
			differentRe: differentRe{re: "invlid[", shouldPanic: true, mustCompile: true},
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

			if tt.re != "" {
				defer func() {
					if tt.shouldPanic {
						if r := recover(); r == nil {
							t.Error("Should have panic")
						}
					}
				}()

				if tt.mustCompile {
					reGroup = MustCompile(tt.re)
				} else {
					reGroup, err = Compile(tt.re)
					if err != nil {
						isErrorMatch(t, tt.wantErr, err)
						return
					}
				}
			}

			target := getTarget(t, tt.expected)
			if err = reGroup.MatchToTarget(tt.s, target); err != nil || tt.wantErr != nil {
				isErrorMatch(t, tt.wantErr, err)
				return
			}
			assert.Equal(t, tt.expected, target)
		})
	}
}

func TestGroups(t *testing.T) {
	r := MustCompile(`(?P<duration>.*?)\s+(?P<num>\d+)\s*(?P<str>.*)?`)
	tests := []struct {
		name     string
		s        string
		wantErr  error
		expected map[string]string
	}{
		{
			name:     "Single match",
			s:        "5s 123 foo",
			wantErr:  nil,
			expected: map[string]string{"duration": "5s", "num": "123", "str": "foo"},
		},
		{
			name:     "No match",
			s:        "5s aa foo",
			wantErr:  &NoMatchFoundError{},
			expected: nil,
		},
		{
			name:     "Empty group",
			s:        "5s 123",
			wantErr:  nil,
			expected: map[string]string{"duration": "5s", "num": "123", "str": ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups, err := r.Groups(tt.s)
			if err != nil || tt.wantErr != nil {
				isErrorMatch(t, tt.wantErr, err)
				return
			}

			require.Equal(t, tt.expected, groups)
		})
	}
}

func TestMatchAllToTarget(t *testing.T) {
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
			target := getTarget(t, tt.expected[0])
			matches, err := r.MatchAllToTarget(tt.s, -1, target)
			if err != nil || tt.wantErr != nil {
				isErrorMatch(t, tt.wantErr, err)
				return
			}
			require.Len(t, matches, len(tt.expected))
			for i, match := range matches {
				assert.Equalf(t, tt.expected[i], match, "Not equal match at index %d", i)
			}
		})
	}
}

func TestBooleanExistenceCheck(t *testing.T) {
	type Exist struct {
		IsAdmin bool `regroup:"is_admin,exists"`
	}
	r := MustCompile(`^(?P<name>\w*)(?:,(?P<is_admin>admin))?$`)
	tests := map[string]struct {
		inputString string
		testFunc    func(t *testing.T, parsed *Exist, err error)
	}{
		"present flag": {
			inputString: "bob_smith,admin",
			testFunc: func(t *testing.T, parsed *Exist, err error) {
				assert.NoError(t, err)
				assert.True(t, parsed.IsAdmin)
			},
		},
		"misspelled flag": {
			inputString: "bob_smith,bladmin",
			testFunc: func(t *testing.T, parsed *Exist, err error) {
				assert.Error(t, err)
			},
		},
		"no flag": {
			inputString: "bob_smith",
			testFunc: func(t *testing.T, parsed *Exist, err error) {
				assert.NoError(t, err)
				assert.False(t, parsed.IsAdmin)
			},
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			parsed := &Exist{}
			err := r.MatchToTarget(tc.inputString, parsed)
			tc.testFunc(t, parsed, err)
		})
	}
}
