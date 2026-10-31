// Copyright 2026 The ggwd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cocoa

import (
	"reflect"
	"runtime"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// objc.ID.Send boxes its arguments in interfaces and calls through reflection, which allocates
// several objects per call. The methods that run every frame call objc_msgSend with MsgSend
// instead.
var objcMsgSend uintptr

// msgSendArgs holds the arguments of a MsgSend call. purego.SyscallN lets its arguments escape,
// so a buffer on the stack would be moved to the heap on every call.
var msgSendArgs = sync.Pool{New: func() any { return new([maxMsgSendArgs]uintptr) }}

// maxMsgSendArgs is the most arguments purego.SyscallN takes.
const maxMsgSendArgs = 15

func init() {
	lib, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	objcMsgSend, err = purego.Dlsym(lib, "objc_msgSend")
	if err != nil {
		panic(err)
	}
}

// MsgSend0 to MsgSend5 send sel to id with as many arguments, and return the integer or pointer
// result. The arguments must be integers, pointers, or structs that the ABI passes in integer
// registers or by reference; floating-point arguments are not supported. Each arity has a
// function of its own, as a variadic slice of arguments would escape to the heap on every call.
//
// A pointer converted to uintptr in the call's arguments stays valid until the function returns.

//go:uintptrescapes
func MsgSend0(id objc.ID, sel objc.SEL) uintptr {
	return msgSend(2, uintptr(id), uintptr(sel), 0, 0, 0, 0, 0)
}

//go:uintptrescapes
func MsgSend1(id objc.ID, sel objc.SEL, a1 uintptr) uintptr {
	return msgSend(3, uintptr(id), uintptr(sel), a1, 0, 0, 0, 0)
}

//go:uintptrescapes
func MsgSend2(id objc.ID, sel objc.SEL, a1, a2 uintptr) uintptr {
	return msgSend(4, uintptr(id), uintptr(sel), a1, a2, 0, 0, 0)
}

//go:uintptrescapes
func MsgSend3(id objc.ID, sel objc.SEL, a1, a2, a3 uintptr) uintptr {
	return msgSend(5, uintptr(id), uintptr(sel), a1, a2, a3, 0, 0)
}

//go:uintptrescapes
func MsgSend4(id objc.ID, sel objc.SEL, a1, a2, a3, a4 uintptr) uintptr {
	return msgSend(6, uintptr(id), uintptr(sel), a1, a2, a3, a4, 0)
}

//go:uintptrescapes
func MsgSend5(id objc.ID, sel objc.SEL, a1, a2, a3, a4, a5 uintptr) uintptr {
	return msgSend(7, uintptr(id), uintptr(sel), a1, a2, a3, a4, a5)
}

// msgSend calls objc_msgSend with the first n of id, sel and a1 to a5.
func msgSend(n int, id, sel, a1, a2, a3, a4, a5 uintptr) uintptr {
	buf := msgSendArgs.Get().(*[maxMsgSendArgs]uintptr)
	buf[0], buf[1], buf[2], buf[3], buf[4], buf[5], buf[6] = id, sel, a1, a2, a3, a4, a5
	r, _, _ := purego.SyscallN(objcMsgSend, buf[:n]...)
	msgSendArgs.Put(buf)
	return r
}

// objc.Send registers, through reflection, the function that sends the message at every call:
// a dozen allocations for a window's size asked every frame. Send registers it once for each
// result type, and SendRect and SendRectRect, for the rectangles asked every frame, take and
// give one without boxing it.
var (
	sends sync.Map // a result's reflect.Type to its func(objc.ID, objc.SEL, ...any) T

	// objcMsgSendStret is objc_msgSend for a struct result returned in memory, on amd64 alone.
	objcMsgSendStret uintptr

	sendRect     func(id objc.ID, sel objc.SEL) NSRect
	sendRectRect func(id objc.ID, sel objc.SEL, r NSRect) NSRect
)

func init() {
	sym := objcMsgSend
	if runtime.GOARCH == "amd64" {
		lib, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			panic(err)
		}
		if objcMsgSendStret, err = purego.Dlsym(lib, "objc_msgSend_stret"); err != nil {
			panic(err)
		}
		sym = objcMsgSendStret // an NSRect, 32 bytes, is returned in memory
	}
	purego.RegisterFunc(&sendRect, sym)
	purego.RegisterFunc(&sendRectRect, sym)
}

// Send is objc.Send, the function that sends the message made once for T.
func Send[T any](id objc.ID, sel objc.SEL, args ...any) T {
	t := reflect.TypeFor[T]()
	f, ok := sends.Load(t)
	if !ok {
		var fn func(objc.ID, objc.SEL, ...any) T
		sym := objcMsgSend
		// purego's register allocation limit for a struct result, as objc.Send decides.
		if runtime.GOARCH == "amd64" && t.Kind() == reflect.Struct && t.Size() > 16 {
			sym = objcMsgSendStret
		}
		purego.RegisterFunc(&fn, sym)
		f, _ = sends.LoadOrStore(t, fn)
	}
	return f.(func(objc.ID, objc.SEL, ...any) T)(id, sel, args...)
}

// SendRect sends sel, which takes no argument, to id for an NSRect.
func SendRect(id objc.ID, sel objc.SEL) NSRect { return sendRect(id, sel) }

// SendRectRect sends sel to id with r for an NSRect.
func SendRectRect(id objc.ID, sel objc.SEL, r NSRect) NSRect { return sendRectRect(id, sel, r) }
