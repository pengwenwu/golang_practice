package geerpc

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

type methodType struct {
	method    reflect.Method // 方法本身
	ArgType   reflect.Type   // 第一个参数的类型
	ReplyType reflect.Type   // 第二个参数的类型
	numCalls  uint64
}

func (t *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&t.numCalls)
}

func (t *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	// arg may be a pointer type, or a value type
	if t.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(t.ArgType.Elem())
	} else {
		argv = reflect.New(t.ArgType).Elem()
	}
	return argv
}

func (t *methodType) newReply() reflect.Value {
	// reply must be a pointer type
	replyv := reflect.New(t.ReplyType.Elem())
	switch t.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(t.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(t.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type service struct {
	name   string        // 结构体名称
	typ    reflect.Type  // 结构体类型
	rcvr   reflect.Value // 结构体实例本身
	method map[string]*methodType
}

func newService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	s.registerMethods()
	return s
}

// registerMethods 过滤出符合条件的方法：
// - 两个导出或内置类型的入参（反射时为3个，第0个是自身，类似self、this）
// - 返回值有且只有1个，类型为error
func (s *service) registerMethods() {
	s.method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: register %s.%s\n", s.name, method.Name)
	}
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

// 能够通过翻设置调用方法
func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	f := m.method.Func
	retutnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := retutnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
