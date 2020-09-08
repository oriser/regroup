package regroup

import (
	"reflect"
	"strconv"
)

type parseFunc func(src string, typ reflect.Type) (reflect.Value, error)

var parsingFuncs = map[reflect.Kind]parseFunc {
	reflect.Int: parseInt,
	reflect.Int8: parseInt,
	reflect.Int16: parseInt,
	reflect.Int64: parseInt,
	reflect.Uint: parseUInt,
	reflect.Uint8: parseUInt,
	reflect.Uint16: parseUInt,
	reflect.Uint64: parseUInt,
	reflect.String: parseString,
}

func parseString(src string, typ reflect.Type) (reflect.Value, error) {
	return reflect.ValueOf(src).Convert(typ), nil
}

func parseInt(src string, typ reflect.Type) (reflect.Value, error) {
	n, err := strconv.ParseInt(src, 10, 64)
	if err != nil {
		return reflect.Value{}, err
	}

	return reflect.ValueOf(n).Convert(typ), nil
}

func parseUInt(src string, typ reflect.Type) (reflect.Value, error) {
	n, err := strconv.ParseUint(src, 10, 64)
	if err != nil {
		return reflect.Value{}, err
	}

	return reflect.ValueOf(n).Convert(typ), nil
}