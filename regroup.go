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

type CompileError error
type NotStructPtrError struct{}

type NoMatchFoundError struct {}
func (n *NoMatchFoundError) Error() string {
	return "no match found for given string"
}

func (n *NotStructPtrError) Error() string {
	return "expected struct pointer"
}

type UnknownGroupError struct {group string}
func (u *UnknownGroupError) Error() string {
	return fmt.Sprintf("group %s didn't found in regex", u.group)
}

type TypeNotParsableError struct{typ reflect.Type}
func (t *TypeNotParsableError) Error() string {
	return fmt.Sprintf("type %v is not parsable", t.typ)
}

type ParseError struct{err error}
func (p *ParseError) Error() string {
	return fmt.Sprintf("parse error: %v", p.err)
}

type RequiredGroupIsEmpty struct{groupName string; fieldName string}
func (r *RequiredGroupIsEmpty) Error() string {
	return fmt.Sprintf("required regroup %s is empty for field %s", r.groupName, r.fieldName)
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
	regroupKey, regroupOption := r.groupAndOption(fieldType)
	if regroupKey == "" {
		return nil
	}
	fieldRefType := fieldType.Type

	matchedVal, ok := matchGroup[regroupKey]
	if !ok {
		return &UnknownGroupError{group: regroupKey}
	}

	if matchedVal == "" && RequiredOption == regroupOption {
		return &RequiredGroupIsEmpty{groupName: regroupKey, fieldName: fieldType.Name}
	}

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

	parsedFunc := getParsingFunc(fieldRefType)
	if parsedFunc == nil {
		return &TypeNotParsableError{fieldRefType}
	}

	parsed, err := parsedFunc(matchedVal, fieldRefType)
	if err != nil {
		return &ParseError{err: err}
	}

	fieldRef.Set(parsed)

	return nil
}

func (r *ReGroup) fillTarget(matchGroup map[string]string, target interface{}) error {
	targetPtr := reflect.ValueOf(target)
	if targetPtr.Kind() != reflect.Ptr {
		return &NotStructPtrError{}
	}
	targetRef := targetPtr.Elem()
	if targetRef.Kind() != reflect.Struct {
		return &NotStructPtrError{}
	}

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

func (r *ReGroup) FindStringSubmatch(s string, target interface{}) error {
	match := r.matcher.FindStringSubmatch(s)
	if match == nil {
		return &NoMatchFoundError{}
	}

	return r.fillTarget(r.matchGroupMap(match), target)
}
