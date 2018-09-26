package mapping

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
)

// ReflectorFunc ...
type ReflectorFunc func(string, interface{}, *map[string]interface{}, *map[uintptr]struct{}) error

var nonAlphaNum = regexp.MustCompile(`[^\w\d*]|[^*]*(\.|\])`)

func refType(iv interface{}) reflect.Type {
	v := reflect.TypeOf(iv)
	if v.Kind() == reflect.Ptr {
		return v.Elem()
	}
	return v
}

func refUnwrap(iv interface{}) reflect.Value {
	v := reflect.ValueOf(iv)
	if v.Kind() == reflect.Ptr {
		return reflect.Indirect(v)
	}
	return v
}

func fieldsVal(v reflect.Value) []reflect.Value {
	fields := make([]reflect.Value, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		fields[i] = v.Field(i)
	}
	return fields
}

func fieldsType(v reflect.Type) []reflect.StructField {
	fields := make([]reflect.StructField, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		fields[i] = v.Field(i)
	}
	return fields
}

// Reflection takes interface as parameter and map it to result map. Be carefull! This function can handle recusive pointers only if you pass argument as a pointer
func Reflection(v interface{}) (refs map[string]interface{}, err error) {
	defer func() {
		if ie := recover(); ie != nil {
			err = errors.New(fmt.Sprint(ie))
		}
	}()
	refs = make(map[string]interface{}, 0)
	pointers := map[uintptr]struct{}{}
	err = getReflection("", v, &refs, &pointers)
	return
}

func getReflection(path string, v interface{}, res *map[string]interface{}, pts *map[uintptr]struct{}) error {
	rType := refType(v)
	rValue := refUnwrap(v)
	if refUnwrap(v).CanAddr() {
		(*pts)[rValue.Addr().Pointer()] = struct{}{}
	}
	switch rType.Kind() {
	case reflect.Struct:
		if err := reflectStruct(path, v, res, pts, getReflection); err != nil {
			return err
		}
	case reflect.Slice:
		if err := reflectSlice(path, v, res, pts, getReflection); err != nil {
			return err
		}
	case reflect.Map:
		if err := reflectMap(path, v, res, pts, getReflection); err != nil {
			return err
		}
	case reflect.Chan:
	default:
		(*res)[toPath(path, "", rValue.Type().String())] = rValue.Interface()
	}
	return nil
}

func reflectStruct(path string, v interface{}, res *map[string]interface{}, pts *map[uintptr]struct{}, rf ReflectorFunc) error {
	rType := refType(v)
	rValue := refUnwrap(v)
	for i := 0; i < rType.NumField(); i++ {
		dstTypeField := rType.Field(i)
		fieldVal := rValue.Field(i)
		kind := dstTypeField.Type.Kind()
		tp := nonAlphaNum.ReplaceAllString(dstTypeField.Type.String(), "")
		switch kind {
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Bool, reflect.Float32, reflect.Float64, reflect.Invalid,
			reflect.Complex64, reflect.Complex128, reflect.Func, reflect.String:
			(*res)[toPath(path, dstTypeField.Name, kind.String())] = fieldVal.Interface()
		case reflect.Ptr:
			if fieldVal.Elem().Kind() == reflect.Invalid {
				(*res)[toPath(path, dstTypeField.Name, tp)] = fieldVal.Interface()
				break
			}
			if _, ok := (*pts)[fieldVal.Elem().Addr().Pointer()]; ok {
				return errors.New("found recursive pointer")
			}
			if err := getReflection(toPath(path, dstTypeField.Name, ""), fieldVal.Interface(), res, pts); err != nil {
				return err
			}
		case reflect.Slice:
			if err := reflectSlice(toPath(path, dstTypeField.Name, ""), fieldVal.Interface(), res, pts, getReflection); err != nil {
				return err
			}
		case reflect.Map:
			if err := reflectMap(toPath(path, dstTypeField.Name, ""), fieldVal.Interface(), res, pts, getReflection); err != nil {
				return err
			}
		}
	}
	return nil
}

func reflectMap(path string, v interface{}, res *map[string]interface{}, pts *map[uintptr]struct{}, rf ReflectorFunc) error {
	rMapVal := refUnwrap(v)
	keys := rMapVal.MapKeys()
	for i := 0; i < len(keys); i++ {
		k := keys[i]
		keyType := k.Type().String()
		val := rMapVal.MapIndex(k)
		fieldName := path + fmt.Sprintf("[%v<%s>]", k.Interface(), keyType)
		switch val.Type().Kind() {
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Bool, reflect.Float32, reflect.Float64, reflect.Invalid,
			reflect.Complex64, reflect.Complex128, reflect.String:
			(*res)[toPath("", fieldName, val.Type().Kind().String())] = val.Interface()
		case reflect.Ptr:
			if val.Elem().Kind() == reflect.Invalid {
				(*res)[toPath("", fieldName, val.Type().Kind().String())] = val.Interface()
				break
			}
			if _, ok := (*pts)[val.Elem().Addr().Pointer()]; ok {
				return errors.New("found recursive pointer")
			}
			if err := rf(toPath("", fieldName, ""), val.Interface(), res, pts); err != nil {
				return err
			}
		case reflect.Interface:
			if err := rf(toPath("", fieldName, ""), val.Interface(), res, pts); err != nil {
				return err
			}
		case reflect.Slice:
			if err := reflectSlice(toPath("", fieldName, ""), val.Interface(), res, pts, rf); err != nil {
				return err
			}
		case reflect.Map:
			if err := reflectMap(toPath("", fieldName, ""), val.Interface(), res, pts, rf); err != nil {
				return err
			}
		}
	}
	return nil
}

func reflectSlice(path string, v interface{}, res *map[string]interface{}, pts *map[uintptr]struct{}, rf ReflectorFunc) error {
	fieldVal := refUnwrap(v)
	for i := 0; i < fieldVal.Len(); i++ {
		val := fieldVal.Index(i)
		fieldName := path + fmt.Sprintf("[%d]", i)
		tp := nonAlphaNum.ReplaceAllString(fieldVal.Type().String(), "")
		switch val.Type().Kind() {
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Bool, reflect.Float32, reflect.Float64, reflect.Invalid,
			reflect.Complex64, reflect.Complex128, reflect.Func, reflect.String:

			(*res)[toPath("", fieldName, tp)] = fieldVal.Index(i).Interface()
		case reflect.Ptr:
			if val.Elem().Kind() == reflect.Invalid {
				(*res)[toPath("", fieldName, tp)] = val.Interface()
				break
			}
			if _, ok := (*pts)[val.Elem().Addr().Pointer()]; ok {
				return errors.New("found recursive pointer")
			}
			if err := getReflection(toPath("", fieldName, ""), val.Interface(), res, pts); err != nil {
				return err
			}

		case reflect.Interface:
			if err := getReflection(toPath("", fieldName, ""), val.Interface(), res, pts); err != nil {
				return err
			}
		case reflect.Slice:
			if err := reflectSlice(toPath("", fieldName, ""), val.Interface(), res, pts, rf); err != nil {
				return err
			}
		case reflect.Map:
			if err := reflectMap(toPath("", fieldName, ""), val.Interface(), res, pts, rf); err != nil {
				return err
			}
		}
	}
	return nil
}

func toPath(p, name, vType string) string {
	if p == "" {
		if vType == "" {
			return name
		}
		if name == "" {
			return "<" + vType + ">"
		}
		return name + "<" + vType + ">"
	}
	if name == "" {
		if vType == "" {
			return p
		}
		return p + "<" + vType + ">"
	}
	if vType == "" {
		return p + "." + name
	}
	return p + "." + name + "<" + vType + ">"
}
