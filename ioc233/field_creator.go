package ioc233

import (
	"math/rand"
	"reflect"
	"time"
)

// ApplyDefaultProviders 为字段应用默认值提供器
// 支持 map、slice、*rand.Rand 等类型的自动初始化
func ApplyDefaultProviders(field reflect.StructField, fv reflect.Value) bool {
	if !fv.CanSet() {
		return false
	}

	fieldType := field.Type

	// 初始化 map
	if fieldType.Kind() == reflect.Map {
		if fv.IsNil() {
			fv.Set(reflect.MakeMap(fieldType))
			return true
		}
		return false
	}

	// 初始化 slice
	if fieldType.Kind() == reflect.Slice {
		if fv.IsNil() {
			fv.Set(reflect.MakeSlice(fieldType, 0, 0))
			return true
		}
		return false
	}

	// 初始化 *rand.Rand
	if fieldType == reflect.TypeOf((*rand.Rand)(nil)) {
		if fv.IsNil() {
			fv.Set(reflect.ValueOf(rand.New(rand.NewSource(time.Now().UnixNano()))))
			return true
		}
		return false
	}

	return false
}
