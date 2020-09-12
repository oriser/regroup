package regroup

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const RequiredOption = "required"

type ReGroup struct {
	matcher *regexp.Regexp
}

func quote(s string) string {
	if strconv.CanBackquote(s) {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

func Compile(expr string) (*ReGroup, error) {
	matcher, err := regexp.Compile(expr)
	if err != nil {
		return nil, CompileError(err)
	}

	return &ReGroup{matcher: matcher}, nil
}

func MustCompile(expr string) *ReGroup {
	reGroup, err := Compile(expr)
	if err != nil {
		panic(`regroup: Compile(` + quote(expr) + `): ` + err.Error())
	}
	return reGroup
}

func (r *ReGroup) matchGroupMap(match []string) map[string]string {
	ret := make(map[string]string)
	for i, name := range r.matcher.SubexpNames() {
		if i != 0 && name != "" {
			ret[name] = match[i]
		}
	}
	return ret
}

func (r *ReGroup) groupAndOption(fieldType reflect.StructField) (group, option string) {
	regroupKey := fieldType.Tag.Get("regroup")
	if regroupKey == "" {
		return "", ""
	}
	splitted := strings.Split(regroupKey, ",")
	if len(splitted) == 1 {
		return splitted[0], ""
	}
	return splitted[0], strings.ToLower(splitted[1])
}

func (r *ReGroup) setField(fieldType reflect.StructField, fieldRef reflect.Value, matchGroup map[string]string) error {
	fieldRefType := fieldType.Type
	if fieldRefType.Kind() == reflect.Ptr {
		if fieldRef.IsNil() {
			return fmt.Errorf("can't set value to nil pointer in struct field: %s", fieldType.Name)
		}
		fieldRefType = fieldType.Type.Elem()
		fieldRef = fieldRef.Elem()
	}

	if fieldRefType.Kind() == reflect.Struct {
		return r.fillTarget(matchGroup, fieldRef)
	}

	regroupKey, regroupOption := r.groupAndOption(fieldType)
	if regroupKey == "" {
		return nil
	}

	matchedVal, ok := matchGroup[regroupKey]
	if !ok {
		return &UnknownGroupError{group: regroupKey}
	}

	if matchedVal == "" {
		if RequiredOption == regroupOption {
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
	fmt.Printf("Matches: %+v\n", matchGroup)
	targetType := targetRef.Type()
	for i := 0; i < targetType.NumField(); i++ {
		fieldRef := targetRef.Field(i)
		if !fieldRef.CanSet() {
			fmt.Printf("Can't set %s\n", targetType.Field(i).Name)
			continue
		}

		if err := r.setField(targetType.Field(i), fieldRef, matchGroup); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReGroup) validateTarget(target interface{}) (reflect.Value, error) {
	targetPtr := reflect.ValueOf(target)
	if targetPtr.Kind() != reflect.Ptr {
		return reflect.Value{}, &NotStructPtrError{}
	}
	return targetPtr.Elem(), nil
}

func (r *ReGroup) MatchToTarget(s string, target interface{}) error {
	match := r.matcher.FindStringSubmatch(s)
	if match == nil {
		return &NoMatchFoundError{}
	}
	fmt.Printf("Match: %+v\n", match)

	targetRef, err := r.validateTarget(target)
	if err != nil {
		return err
	}
	return r.fillTarget(r.matchGroupMap(match), targetRef)
}

func (r *ReGroup) MatchAllToTarget(s string, n int, targetType interface{}) ([]interface{}, error) {
	targetRefType, err := r.validateTarget(targetType)
	if err != nil {
		return nil, err
	}

	matches := r.matcher.FindAllStringSubmatch(s, n)
	fmt.Printf("All matches: %+v\n", matches)
	if matches == nil {
		return nil, &NoMatchFoundError{}
	}

	ret := make([]interface{}, len(matches))
	for i, match := range matches {
		target := reflect.New(targetRefType.Type()).Elem()
		if err := r.fillTarget(r.matchGroupMap(match), target); err != nil {
			return nil, err
		}
		ret[i] = target.Addr().Interface()
	}

	return ret, nil
}
