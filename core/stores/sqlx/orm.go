package sqlx

import (
	"errors"
	"reflect"
	"strings"

	"github.com/zeromicro/go-zero/core/mapping"
)

const tagName = "db"

var (
	// ErrNotMatchDestination is an error that indicates not matching destination to scan.
	ErrNotMatchDestination = errors.New("not matching destination to scan")
	// ErrNotReadableValue is an error that indicates value is not addressable or interfaceable.
	ErrNotReadableValue = errors.New("value not addressable or interfaceable")
	// ErrNotSettable is an error that indicates the passed in variable is not settable.
	ErrNotSettable = errors.New("passed in variable is not settable")
	// ErrUnsupportedValueType is an error that indicates unsupported unmarshal type.
	ErrUnsupportedValueType = errors.New("unsupported unmarshal type")
)

type rowsScanner interface {
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(v ...any) error
}

func getTaggedFieldValueMap(v reflect.Value) (map[string]any, error) {
	rt := mapping.Deref(v.Type())
	size := rt.NumField()
	result := make(map[string]any, size)

	for i := 0; i < size; i++ {
		field := rt.Field(i) //Anonymous 表示是不是嵌入成员
		if field.Anonymous && mapping.Deref(field.Type).Kind() == reflect.Struct {
			// reflect.Indirect(v).Field(i) v要是指针，取出v.Elem(),并得到第i位置的reflect.Value
			inner, err := getTaggedFieldValueMap(reflect.Indirect(v).Field(i)) //回调函数：
			if err != nil {
				return nil, err
			}

			for key, val := range inner {
				result[key] = val
			}

			continue
		}

		key := parseTagName(field)
		if len(key) == 0 {
			continue
		}

		valueField := reflect.Indirect(v).Field(i)
		valueData, err := getValueInterface(valueField)
		if err != nil {
			return nil, err
		}

		result[key] = valueData
	}

	return result, nil
}

// 此函数获取value的值的
func getValueInterface(value reflect.Value) (any, error) {
	switch value.Kind() {
	case reflect.Ptr:
		if !value.CanInterface() { //CanInterface reports whether Interface can be used without panicking.
			return nil, ErrNotReadableValue
		}

		if value.IsNil() { //处理空指针，空指针这里分配空间
			baseValueType := mapping.Deref(value.Type())
			value.Set(reflect.New(baseValueType)) //给一个默认值
		}

		return value.Interface(), nil //Interface returns v's current value as an interface{}
	default:
		if !value.CanAddr() || !value.Addr().CanInterface() {
			return nil, ErrNotReadableValue
		}

		return value.Addr().Interface(), nil
	}
}

// 这里v如果是个结构体，这个函数是依次获取结构体成员的列表，使用row.Scan（...）给结构体成员赋值
func mapStructFieldsIntoSlice(v reflect.Value, columns []string, strict bool) ([]any, error) {
	fields := unwrapFields(v)                 //得到构成v的各成员的value，v此时应该是zero值
	if strict && len(columns) < len(fields) { //列数跟结构体field个数对不上
		return nil, ErrNotMatchDestination
	}

	taggedMap, err := getTaggedFieldValueMap(v) //得到v各成员的fieldName-fieldValue map
	if err != nil {
		return nil, err
	}

	values := make([]any, len(columns))
	if len(taggedMap) == 0 {
		if len(fields) < len(values) {
			return nil, ErrNotMatchDestination
		}

		for i := 0; i < len(values); i++ {
			valueField := fields[i]
			valueData, err := getValueInterface(valueField)
			if err != nil {
				return nil, err
			}

			values[i] = valueData //分配空间了
		}
	} else {
		for i, column := range columns {
			if tagged, ok := taggedMap[column]; ok {
				values[i] = tagged
			} else {
				var anonymous any
				values[i] = &anonymous //分配空间了
			}
		}
	}

	return values, nil
}

func parseTagName(field reflect.StructField) string {
	key := field.Tag.Get(tagName)
	if len(key) == 0 {
		return ""
	}

	options := strings.Split(key, ",")
	return strings.TrimSpace(options[0])
}

func unmarshalRow(v any, scanner rowsScanner, strict bool) error {
	if !scanner.Next() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return ErrNotFound
	}

	rv := reflect.ValueOf(v)
	if err := mapping.ValidatePtr(rv); err != nil {
		return err
	}

	rte := reflect.TypeOf(v).Elem()
	rve := rv.Elem()
	switch rte.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		if !rve.CanSet() {
			return ErrNotSettable
		}

		return scanner.Scan(v)
	case reflect.Struct:
		columns, err := scanner.Columns()
		if err != nil {
			return err
		}

		values, err := mapStructFieldsIntoSlice(rve, columns, strict)
		if err != nil {
			return err
		}

		return scanner.Scan(values...)
	default:
		return ErrUnsupportedValueType
	}
}

// rows *sql.Rows是owsScanner类型，这里v是要反序列化对象，要求v是指针类型且不能是nil 指针，必须分配空间
func unmarshalRows(v any, scanner rowsScanner, strict bool) error {
	rv := reflect.ValueOf(v)
	if err := mapping.ValidatePtr(rv); err != nil {
		return err
	}

	rt := reflect.TypeOf(v) //获取v的类型
	rte := rt.Elem()        //获取v真实的类型
	rve := rv.Elem()        //获取v真实的value
	if !rve.CanSet() {
		return ErrNotSettable
	}

	switch rte.Kind() {
	case reflect.Slice:
		ptr := rte.Elem().Kind() == reflect.Ptr //判断切片的元素是不是指针
		appendFn := func(item reflect.Value) {
			if ptr {
				rve.Set(reflect.Append(rve, item))
			} else {
				rve.Set(reflect.Append(rve, reflect.Indirect(item)))
			}
		}
		fillFn := func(value any) error {
			if err := scanner.Scan(value); err != nil {
				return err
			}

			appendFn(reflect.ValueOf(value)) //执行函数
			return nil
		}

		base := mapping.Deref(rte.Elem()) //切片的元素
		switch base.Kind() {
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.String:
			for scanner.Next() {
				value := reflect.New(base)
				if err := fillFn(value.Interface()); err != nil {
					return err
				}
			}
		case reflect.Struct: //切片的元素是个结构体
			columns, err := scanner.Columns()
			if err != nil {
				return err
			}
			/*
					var option string
				   err := rows.Scan(&option)
			*/

			for scanner.Next() {
				value := reflect.New(base)                                      //构造一个结构体指针，New returns a Value representing a pointer to a new zero value
				values, err := mapStructFieldsIntoSlice(value, columns, strict) //value此时是个结构体指针，把结构体各字段展开为字段指针，好方便row.Scan使用
				if err != nil {
					return err
				}

				if err := scanner.Scan(values...); err != nil {
					return err
				}

				appendFn(value)
			}
		default:
			return ErrUnsupportedValueType
		}

		return nil
	default:
		return ErrUnsupportedValueType
	}
}

// 把大的reflect.Value值拆开成每个filed的reflect.Value构成的列表
func unwrapFields(v reflect.Value) []reflect.Value {
	var fields []reflect.Value
	indirect := reflect.Indirect(v)

	for i := 0; i < indirect.NumField(); i++ {
		child := indirect.Field(i)
		if !child.CanSet() {
			continue
		}

		if child.Kind() == reflect.Ptr && child.IsNil() {
			baseValueType := mapping.Deref(child.Type())
			child.Set(reflect.New(baseValueType))
		}

		child = reflect.Indirect(child)
		childType := indirect.Type().Field(i)
		if child.Kind() == reflect.Struct && childType.Anonymous {
			fields = append(fields, unwrapFields(child)...)
		} else {
			fields = append(fields, child)
		}
	}

	return fields
}
