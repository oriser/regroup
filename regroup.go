package regroup

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

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

		fieldType := targetType.Field(i)
		regroupKey := fieldType.Tag.Get("regroup")
		if regroupKey == "" {
			continue
		}

		matchedVal, ok := matchGroup[regroupKey]
		if !ok {
			return &UnknownGroupError{group: regroupKey}
		}

		parsedFunc, ok := parsingFuncs[fieldType.Type.Kind()]
		if !ok {
			return &TypeNotParsableError{fieldType.Type}
		}

		parsed, err := parsedFunc(matchedVal, fieldType.Type)
		if err != nil {
			return &ParseError{err: err}
		}

		fieldRef.Set(parsed)
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
