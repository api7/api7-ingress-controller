package types

import (
	"fmt"
	"reflect"
	"sync"

	"google.golang.org/protobuf/proto"
)

type Map[K comparable, V any] struct {
	value map[K]V

	lock sync.RWMutex
}

func (m *Map[K, V]) init() {
	if m.value == nil {
		m.value = make(map[K]V)
	}
}

func (m *Map[K, V]) LoadAllOrDelete() map[K]V {
	m.init()

	m.lock.Lock()
	defer m.lock.Unlock()

	ret := make(map[K]V, len(m.value))
	for k, v := range m.value {
		ret[k] = DeepCopy(v)
		delete(m.value, k)
	}
	return ret
}

func (m *Map[K, V]) LoadAll() map[K]V {
	return m.LoadAllMatching(func(K, V) bool {
		return true
	})
}

func (m *Map[K, V]) LoadAllMatching(matching func(K, V) bool) map[K]V {
	m.init()

	m.lock.RLock()
	defer m.lock.RUnlock()

	ret := make(map[K]V, len(m.value))
	for k, v := range m.value {
		if matching(k, v) {
			ret[k] = DeepCopy(v)
		}
	}
	return ret
}

// Len returns the number of key/value pairs in the map.
func (m *Map[K, V]) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.value)
}

// Load returns a deepcopy of the value for a specific key.
func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	ret, ok := m.value[key]
	if !ok {
		return ret, false
	}
	return DeepCopy(ret), true
}

func (m *Map[K, V]) Store(key K, val V) {
	m.init()

	m.lock.Lock()
	defer m.lock.Unlock()

	m.value[key] = DeepCopy(val)
}

func (m *Map[K, V]) LoadOrStore(key K, val V) (value V, loaded bool) {
	m.init()

	m.lock.Lock()
	defer m.lock.Unlock()

	v, ok := m.value[key]
	if ok {
		return DeepCopy(v), true
	}
	m.value[key] = DeepCopy(val)
	return DeepCopy(val), false
}

func (m *Map[K, V]) Delete(key K) {
	m.init()

	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.value, key)
}

func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	m.init()

	m.lock.Lock()
	defer m.lock.Unlock()

	v, ok := m.value[key]
	if !ok {
		return v, false
	}

	delete(m.value, key)
	return DeepCopy(v), true
}

type (
	_DeepCopier[T any] interface {
		DeepCopy() T
	}
	DeepCopier[T _DeepCopier[T]] _DeepCopier[T]
)

func hasNoPointers(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true
	case reflect.Array:
		return hasNoPointers(typ.Elem())
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			if !hasNoPointers(typ.Field(i).Type) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func DeepCopy[T any](val T) T {
	switch tval := any(val).(type) {
	case _DeepCopier[T]:
		return tval.DeepCopy()
	case proto.Message:
		return proto.Clone(tval).(T)
	default:
		if hasNoPointers(reflect.TypeOf(val)) {
			return val
		}
		panic(fmt.Errorf("watchable.DeepCopy: type is not copiable: %T", val))
	}
}

type (
	_Comparable[T any] interface {
		Equal(T) bool
	}
	Comparable[T _Comparable[T]] _Comparable[T]
)

func DeepEqual[T any](a, b T) bool {
	switch ta := any(a).(type) {
	case _Comparable[T]:
		return ta.Equal(b)
	case proto.Message:
		if tb, ok := any(b).(proto.Message); ok {
			return proto.Equal(ta, tb)
		}
		return reflect.DeepEqual(a, b)
	default:
		return reflect.DeepEqual(a, b)
	}
}
