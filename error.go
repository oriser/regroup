package regroup

import (
	"fmt"
	"reflect"
)

type CompileError error

type NoMatchFoundError struct{}

func (n *NoMatchFoundError) Error() string {
	return "no match found for given string"
}

type NotStructPtrError struct{}

func (n *NotStructPtrError) Error() string {
	return "expected struct pointer"
}

type UnknownGroupError struct{ group string }

func (u *UnknownGroupError) Error() string {
	return fmt.Sprintf("group %s didn't found in regex", u.group)
}

type TypeNotParsableError struct{ typ reflect.Type }

func (t *TypeNotParsableError) Error() string {
	return fmt.Sprintf("type %v is not parsable", t.typ)
}

type ParseError struct {
	group string
	err   error
}

func (p *ParseError) Error() string {
	return fmt.Sprintf("error parsing group %s: %v", p.group, p.err)
}

type RequiredGroupIsEmpty struct {
	groupName string
	fieldName string
}

func (r *RequiredGroupIsEmpty) Error() string {
	return fmt.Sprintf("required regroup %s is empty for field %s", r.groupName, r.fieldName)
}
