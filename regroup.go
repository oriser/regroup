package regroup

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

const (
	requiredOption = "required"
	existsOption   = "exists"
)

// ReGroup is the main ReGroup matcher struct
type ReGroup struct {
	matcher *regexp.Regexp
}

func quote(s string) string {
	if strconv.CanBackquote(s) {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

// Compile compiles given expression as regex and return new ReGroup with this expression as matching engine.
// If the expression can't be compiled as regex, a CompileError will be returned
func Compile(expr string) (*ReGroup, error) {
	matcher, err := regexp.Compile(expr)
	if err != nil {
		return nil, &CompileError{err: err}
	}

	return &ReGroup{matcher: matcher}, nil
}

// MustCompile calls Compile and panics if it returns an error
func MustCompile(expr string) *ReGroup {
	reGroup, err := Compile(expr)
	if err != nil {
		panic(`regroup: Compile(` + quote(expr) + `): ` + err.Error())
	}
	return reGroup
}

// matchGroupMap converts the match string array into a map of group keys to group values
func (r *ReGroup) matchGroupMap(match []string) map[string]string {
	ret := make(map[string]string)
	for i, name := range r.matcher.SubexpNames() {
		if i != 0 && name != "" {
			ret[name] = match[i]
		}
	}
	return ret
}

// groupAndOption returns the requested regroup and its options split by ','
func (r *ReGroup) groupAndOption(fieldType reflect.StructField) (group string, option []string) {
	regroupKey := fieldType.Tag.Get("regroup")
	if regroupKey == "" {
		return "", nil
	}
	split := strings.Split(regroupKey, ",")
	if len(split) == 1 {
		return strings.TrimSpace(split[0]), nil
	}
	var options []string
	for _, opt := range split[1:] {
		options = append(options, strings.TrimSpace(opt))
	}
	return strings.TrimSpace(split[0]), options
}

// setField getting a single struct field and matching groups map and set the field value to its matching group value tag
// after parsing it to match the field type
func (r *ReGroup) setField(fieldType reflect.StructField, fieldRef reflect.Value, matchGroup map[string]string) error {
	fieldRefType := fieldType.Type
	ptr := false
	if fieldRefType.Kind() == reflect.Ptr {
		ptr = true
		fieldRefType = fieldType.Type.Elem()
	}

	if fieldRefType.Kind() == reflect.Struct {
		if ptr {
			if fieldRef.IsNil() {
				return fmt.Errorf("can't set value to nil pointer in struct field: %s", fieldType.Name)
			}
			fieldRef = fieldRef.Elem()
		}
		if fieldRefType.Name() == "Time" && fieldRefType.PkgPath() == "time" {
			return r.fillTimeTarget(fieldType, fieldRef, matchGroup)
		}
		return r.fillTarget(matchGroup, fieldRef)
	}

	regroupKey, regroupOptions := r.groupAndOption(fieldType)
	if regroupKey == "" {
		return nil
	}

	if ptr {
		if fieldRef.IsNil() {
			return fmt.Errorf("can't set value to nil pointer in field: %s", fieldType.Name)
		}
		fieldRef = fieldRef.Elem()
	}

	matchedVal, ok := matchGroup[regroupKey]
	if !ok {
		return &UnknownGroupError{group: regroupKey}
	}

	if slices.Contains(regroupOptions, existsOption) {
		fieldRef.SetBool(matchedVal != "")
		return nil
	}

	if matchedVal == "" {
		if slices.Contains(regroupOptions, requiredOption) {
			return &RequiredGroupIsEmpty{groupName: regroupKey, fieldName: fieldType.Name}
		}
		return nil
	}

	parsedFunc := getParsingFunc(fieldRefType)
	if parsedFunc == nil {
		return &TypeNotParsableError{fieldRefType}
	}

	parsed, err := parsedFunc(matchedVal, fieldRefType)
	if err != nil {
		return &ParseError{group: regroupKey, err: err}
	}

	fieldRef.Set(parsed)

	return nil
}

func (r *ReGroup) fillTarget(matchGroup map[string]string, targetRef reflect.Value) error {
	targetType := targetRef.Type()
	for i := 0; i < targetType.NumField(); i++ {
		fieldRef := targetRef.Field(i)
		if !fieldRef.CanSet() {
			continue
		}

		if err := r.setField(targetType.Field(i), fieldRef, matchGroup); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReGroup) fillTimeTarget(fieldType reflect.StructField, fieldRef reflect.Value, matchGroup map[string]string) error {
	regroupKey, regroupOptions := r.groupAndOption(fieldType)
	if regroupKey == "" {
		return nil
	}
	var pattern string
	if len(regroupOptions) == 0 {
		pattern = time.RFC3339
	} else {
		pattern = regroupOptions[0]
	}
	parsed, err := time.Parse(pattern, matchGroup[regroupKey])
	if err != nil {
		return err
	}
	fieldRef.Set(reflect.ValueOf(parsed))
	return nil
}

// validateTarget checks that given interface is a pointer of struct
func (r *ReGroup) validateTarget(target interface{}) (reflect.Value, error) {
	targetPtr := reflect.ValueOf(target)
	if targetPtr.Kind() != reflect.Ptr {
		return reflect.Value{}, &NotStructPtrError{}
	}
	return targetPtr.Elem(), nil
}

// Groups returns a map contains each group name as a key and the group's matched value as value
func (r *ReGroup) Groups(s string) (map[string]string, error) {
	match := r.matcher.FindStringSubmatch(s)
	if match == nil {
		return nil, &NoMatchFoundError{}
	}

	return r.matchGroupMap(match), nil
}

// MatchToTarget matches a regex expression to string s and parse it into `target` argument.
// If no matches found, a &NoMatchFoundError error will be returned
func (r *ReGroup) MatchToTarget(s string, target interface{}) error {
	match := r.matcher.FindStringSubmatch(s)
	if match == nil {
		return &NoMatchFoundError{}
	}

	targetRef, err := r.validateTarget(target)
	if err != nil {
		return err
	}
	return r.fillTarget(r.matchGroupMap(match), targetRef)
}

// Creating a new pointer to given target type
// Will recurse over all NOT NIL struct pointer and create them too
func (r *ReGroup) newTargetType(originalRef reflect.Value) reflect.Value {
	originalType := originalRef.Type()

	target := reflect.New(originalRef.Type()).Elem()
	for i := 0; i < originalRef.NumField(); i++ {
		origFieldRef := originalRef.Field(i)
		if originalType.Field(i).Type.Kind() == reflect.Ptr {
			if origFieldRef.IsNil() || !target.Field(i).CanSet() {
				continue
			}
			origElem := origFieldRef.Elem()
			if origElem.Type().Kind() == reflect.Struct {
				// If the type is not nil struct pointer, recurse into it to create all necessary fields inside
				target.Field(i).Set(r.newTargetType(origElem).Addr())
			} else {
				target.Field(i).Set(reflect.New(origElem.Type()))
			}

		}
	}
	return target
}

// MatchAllToTarget will find all the regex matches for given string 's',
// and parse them into objects of the same type as `targetType` argument.
// The return type is an array of interfaces, which every element is the same type as `targetType` argument.
// If no matches found, a &NoMatchFoundError error will be returned
func (r *ReGroup) MatchAllToTarget(s string, n int, targetType interface{}) ([]interface{}, error) {
	targetRefType, err := r.validateTarget(targetType)
	if err != nil {
		return nil, err
	}

	matches := r.matcher.FindAllStringSubmatch(s, n)
	if matches == nil {
		return nil, &NoMatchFoundError{}
	}

	ret := make([]interface{}, len(matches))
	for i, match := range matches {
		target := r.newTargetType(targetRefType)
		if err := r.fillTarget(r.matchGroupMap(match), target); err != nil {
			return nil, err
		}
		ret[i] = target.Addr().Interface()
	}

	return ret, nil
}
