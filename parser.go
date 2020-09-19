package regroup

import (
	"reflect"
	"strconv"
	"time"
)

type parseFunc func(src string, typ reflect.Type) (reflect.Value, error)

var builtinTypesParsingFuncs = map[reflect.Kind]parseFunc{
	reflect.Bool:    parseBool,
	reflect.String:  parseString,
	reflect.Int:     parseInt,
	reflect.Int8:    parseInt,
	reflect.Int16:   parseInt,
	reflect.Int32:   parseInt,
	reflect.Int64:   parseInt,
	reflect.Uint:    parseUInt,
	reflect.Uint8:   parseUInt,
	reflect.Uint16:  parseUInt,
	reflect.Uint32:  parseUInt,
	reflect.Uint64:  parseUInt,
	reflect.Float32: parseFloat,
	reflect.Float64: parseFloat,
}

var typesParsingFuncs = map[reflect.Type]parseFunc{
	reflect.TypeOf(time.Second): parseDuration,
}

func getParsingFunc(typ reflect.Type) parseFunc {
	if parsingFunc, ok := typesParsingFuncs[typ]; ok {
		return parsingFunc
	}
	if parsingFunc, ok := builtinTypesParsingFuncs[typ.Kind()]; ok {
		return parsingFunc
	}
	return nil
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

func parseFloat(src string, typ reflect.Type) (reflect.Value, error) {
	n, err := strconv.ParseFloat(src, 64)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(n).Convert(typ), nil
}

func parseBool(src string, _ reflect.Type) (reflect.Value, error) {
	b, err := strconv.ParseBool(src)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(b), nil
}

func parseDuration(src string, _ reflect.Type) (reflect.Value, error) {
	d, err := time.ParseDuration(src)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(d), nil
}
