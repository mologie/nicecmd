package nicecmd

import (
	"github.com/spf13/pflag"
	"reflect"
)

type typeReg struct {
	Name       string
	Parser     func(value string) (any, error)
	Serializer func(value any) string
}

var typeRegs = map[reflect.Type]typeReg{}

// RegisterType registers a custom type for BindConfig. The typical use-case for RegisterType is to
// make third-party types embeddable into configuration structs. Implementing pflag.Value is the
// nicer solution for first-party types, because it does not rely on global state.
func RegisterType[T any](
	parse func(value string) (T, error),
	serialize func(value T) string,
) {
	t := reflect.TypeFor[T]()
	typeRegs[t] = typeReg{
		Name: t.Name(),
		Parser: func(value string) (any, error) {
			return parse(value)
		},
		Serializer: func(value any) string {
			return serialize(value.(T))
		},
	}
}

// UnregisterType removes a custom type registration. Useful for testing.
func UnregisterType[T any]() {
	delete(typeRegs, reflect.TypeFor[T]())
}

func useCustomValue(value reflect.Value) (pflag.Value, bool) {
	if reg, ok := typeRegs[value.Type()]; ok {
		return &typeWrap{value: value, reg: reg}, true
	}
	return nil, false
}

var _ pflag.Value = &typeWrap{}

// typeWrap implements pflag.Value for custom types registered with RegisterType.
type typeWrap struct {
	value reflect.Value
	reg   typeReg
}

func (tw *typeWrap) Set(value string) error {
	v, err := tw.reg.Parser(value)
	if err != nil {
		return err
	}
	tw.value.Set(reflect.ValueOf(v))
	return nil
}

func (tw *typeWrap) String() string {
	return tw.reg.Serializer(tw.value.Interface())
}

func (tw *typeWrap) Type() string {
	return tw.reg.Name
}
