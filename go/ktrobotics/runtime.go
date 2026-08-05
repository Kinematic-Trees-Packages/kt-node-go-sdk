package ktrobotics

/*
#cgo LDFLAGS: -lkt_node
#include <stdlib.h>
#include <stdint.h>
#include "kt_robotics.h"

extern kt_algorithm_outcome_t goKtSetup(void* user_data, kt_algorithm_context_t* context);
extern kt_algorithm_outcome_t goKtStep(void* user_data, kt_algorithm_context_t* context);
extern kt_algorithm_outcome_t goKtClose(void* user_data, kt_algorithm_context_t* context);

static kt_algorithm_setup_fn kt_go_setup_fn(void) { return goKtSetup; }
static kt_algorithm_step_fn kt_go_step_fn(void) { return goKtStep; }
static kt_algorithm_close_fn kt_go_close_fn(void) { return goKtClose; }
static void* kt_go_handle_ptr(uintptr_t value) { return (void*)value; }
static uintptr_t kt_go_handle_value(void* value) { return (uintptr_t)value; }
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime/cgo"
	"unsafe"
)

const ABIMajor = int(C.KT_ABI_VERSION_MAJOR)
const ABIMinor = int(C.KT_ABI_VERSION_MINOR)

type NextStep uint32

const (
	Continue    NextStep = C.KT_ALGORITHM_CONTINUE
	Stop        NextStep = C.KT_ALGORITHM_STOP
	Recoverable NextStep = C.KT_ALGORITHM_RECOVERABLE
	Fatal       NextStep = C.KT_ALGORITHM_FATAL
)

type ReadMode uint32

const (
	ReadOne          ReadMode = C.KT_READ_ONE
	ReadAllAvailable ReadMode = C.KT_READ_ALL_AVAILABLE
	ReadCount        ReadMode = C.KT_READ_COUNT
)

type Error struct {
	Status  uint32
	Message string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("KT status %d", e.Status)
}

type Message struct {
	Payload      []byte
	SourceID     *string
	RemoteTimeNS *int64
}

type Context struct {
	ptr *C.kt_algorithm_context_t
}

type Node interface {
	Setup(*Context) NextStep
	Step(*Context) NextStep
	Close(*Context) NextStep
}

type NodeFunc struct {
	SetupFunc func(*Context) NextStep
	StepFunc  func(*Context) NextStep
	CloseFunc func(*Context) NextStep
}

func (n NodeFunc) Setup(ctx *Context) NextStep {
	if n.SetupFunc != nil {
		return n.SetupFunc(ctx)
	}
	return Continue
}
func (n NodeFunc) Step(ctx *Context) NextStep {
	if n.StepFunc != nil {
		return n.StepFunc(ctx)
	}
	return Stop
}
func (n NodeFunc) Close(ctx *Context) NextStep {
	if n.CloseFunc != nil {
		return n.CloseFunc(ctx)
	}
	return Stop
}

func ABIVersion() (major, minor uint32) {
	return uint32(C.kt_abi_version_major()), uint32(C.kt_abi_version_minor())
}

func RuntimeVersion() (major, minor, patch uint32, err error) {
	var out C.kt_version_v1
	out.struct_size = C.uint32_t(C.sizeof_kt_version_v1)
	out.abi_version = C.KT_ABI_VERSION_MAJOR
	if e := statusError(C.kt_runtime_version(&out), nil); e != nil {
		return 0, 0, 0, e
	}
	return uint32(out.major), uint32(out.minor), uint32(out.patch), nil
}

func BuildID() string { return stringViewToString(C.kt_runtime_build_id()) }

func (c *Context) IsClosing() (bool, error) {
	var out C.uint32_t
	if err := statusError(C.kt_context_is_closing(c.ptr, &out), nil); err != nil {
		return false, err
	}
	return out != 0, nil
}

func (c *Context) RequestClose() error { return statusError(C.kt_context_request_close(c.ptr), nil) }

func (c *Context) ReportError(message string) error {
	view, free := stringView(message)
	defer free()
	return statusError(C.kt_context_report_error(c.ptr, view), nil)
}

func (c *Context) Set(channel string, payload []byte) error {
	ch, freeCh := stringView(channel)
	defer freeCh()
	bytes, freeBytes := bytesView(payload)
	defer freeBytes()
	var ktErr *C.kt_error_t
	return statusError(C.kt_context_set(c.ptr, ch, bytes, &ktErr), ktErr)
}

func (c *Context) SetFrom(channel, sourceID string, payload []byte) error {
	ch, freeCh := stringView(channel)
	defer freeCh()
	src, freeSrc := stringView(sourceID)
	defer freeSrc()
	bytes, freeBytes := bytesView(payload)
	defer freeBytes()
	var ktErr *C.kt_error_t
	return statusError(C.kt_context_set_source(c.ptr, ch, src, bytes, &ktErr), ktErr)
}

func (c *Context) MetricsJSON() ([]byte, error) {
	var out *C.kt_owned_bytes_t
	var ktErr *C.kt_error_t
	if err := statusError(C.kt_context_metrics_json(c.ptr, &out, &ktErr), ktErr); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	defer C.kt_owned_bytes_destroy(&out)
	return bytesViewToBytes(C.kt_owned_bytes_view(out)), nil
}

func (c *Context) Get(channel string, mode ReadMode, count uint64) ([]Message, error) {
	ch, freeCh := stringView(channel)
	defer freeCh()
	options := C.kt_read_options_v1{struct_size: C.uint32_t(C.sizeof_kt_read_options_v1), abi_version: C.KT_ABI_VERSION_MAJOR, mode: C.kt_read_mode_t(mode), count: C.uint64_t(count)}
	var batch *C.kt_message_batch_t
	var ktErr *C.kt_error_t
	if err := statusError(C.kt_context_read(c.ptr, ch, &options, &batch, &ktErr), ktErr); err != nil {
		return nil, err
	}
	if batch == nil {
		return nil, nil
	}
	defer C.kt_message_batch_destroy(&batch)
	total := uint64(C.kt_message_batch_count(batch))
	messages := make([]Message, 0, total)
	for i := uint64(0); i < total; i++ {
		item := C.kt_message_view_v1{struct_size: C.uint32_t(C.sizeof_kt_message_view_v1), abi_version: C.KT_ABI_VERSION_MAJOR}
		var itemErr *C.kt_error_t
		if err := statusError(C.kt_message_batch_item(batch, C.uint64_t(i), &item, &itemErr), itemErr); err != nil {
			return nil, err
		}
		msg := Message{Payload: bytesViewToBytes(item.payload)}
		if item.has_source != 0 {
			s := stringViewToString(item.source_id)
			msg.SourceID = &s
		}
		if item.has_remote_time != 0 {
			t := int64(item.remote_time_ns)
			msg.RemoteTimeNS = &t
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

type Runtime struct {
	ptr    *C.kt_runtime_t
	handle cgo.Handle
	closed bool
}

type runtimeState struct{ node Node }

func NewRuntime(packagePath, runtimePath string, node Node) (*Runtime, error) {
	if node == nil {
		return nil, errors.New("ktrobotics: nil node")
	}
	if C.kt_abi_version_major() != C.KT_ABI_VERSION_MAJOR {
		return nil, fmt.Errorf("unsupported KT Robotics ABI major %d", uint32(C.kt_abi_version_major()))
	}
	state := &runtimeState{node: node}
	handle := cgo.NewHandle(state)
	pkg, freePkg := stringView(packagePath)
	defer freePkg()
	runtimeCfg, freeRuntime := stringView(runtimePath)
	defer freeRuntime()
	callbacks := (*C.kt_algorithm_callbacks_v1)(C.malloc(C.sizeof_kt_algorithm_callbacks_v1))
	if callbacks == nil {
		handle.Delete()
		return nil, errors.New("ktrobotics: failed to allocate callback table")
	}
	defer C.free(unsafe.Pointer(callbacks))
	*callbacks = C.kt_algorithm_callbacks_v1{struct_size: C.uint32_t(C.sizeof_kt_algorithm_callbacks_v1), abi_version: C.KT_ABI_VERSION_MAJOR, setup: C.kt_go_setup_fn(), step: C.kt_go_step_fn(), close: C.kt_go_close_fn()}
	options := C.kt_runtime_options_v1{struct_size: C.uint32_t(C.sizeof_kt_runtime_options_v1), abi_version: C.KT_ABI_VERSION_MAJOR, package_path: pkg, runtime_path: runtimeCfg, callbacks: callbacks, user_data: C.kt_go_handle_ptr(C.uintptr_t(handle))}
	var out *C.kt_runtime_t
	var ktErr *C.kt_error_t
	if err := statusError(C.kt_runtime_create_v1(&options, &out, &ktErr), ktErr); err != nil {
		handle.Delete()
		return nil, err
	}
	return &Runtime{ptr: out, handle: handle}, nil
}

func (r *Runtime) Run() error {
	var ktErr *C.kt_error_t
	return statusError(C.kt_runtime_run(r.ptr, &ktErr), ktErr)
}

func (r *Runtime) RequestClose() error { return statusError(C.kt_runtime_request_close(r.ptr), nil) }

func (r *Runtime) Close() error {
	if r == nil || r.closed {
		return nil
	}
	r.closed = true
	var ktErr *C.kt_error_t
	err := statusError(C.kt_runtime_destroy(&r.ptr, &ktErr), ktErr)
	r.handle.Delete()
	return err
}

func Run(packagePath, runtimePath string, node Node) error {
	r, err := NewRuntime(packagePath, runtimePath, node)
	if err != nil {
		return err
	}
	defer r.Close()
	return r.Run()
}

func stringView(value string) (C.kt_string_view_t, func()) {
	cstr := C.CString(value)
	return C.kt_string_view_t{data: cstr, length: C.uint64_t(len(value))}, func() { C.free(unsafe.Pointer(cstr)) }
}

func bytesView(payload []byte) (C.kt_bytes_view_t, func()) {
	if len(payload) == 0 {
		return C.kt_bytes_view_t{}, func() {}
	}
	ptr := C.CBytes(payload)
	return C.kt_bytes_view_t{data: (*C.uint8_t)(ptr), length: C.uint64_t(len(payload))}, func() { C.free(ptr) }
}

func stringViewToString(view C.kt_string_view_t) string {
	if view.data == nil || view.length == 0 {
		return ""
	}
	return C.GoStringN(view.data, C.int(view.length))
}

func bytesViewToBytes(view C.kt_bytes_view_t) []byte {
	if view.data == nil || view.length == 0 {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(view.data), C.int(view.length))
}

func statusName(status C.kt_status_t) string { return stringViewToString(C.kt_status_name(status)) }

func statusError(status C.kt_status_t, ktErr *C.kt_error_t) error {
	if status == C.KT_STATUS_OK {
		return nil
	}
	message := ""
	if ktErr != nil {
		message = stringViewToString(C.kt_error_message(ktErr))
		C.kt_error_destroy(&ktErr)
	}
	if message == "" {
		message = statusName(status)
	}
	return &Error{Status: uint32(status), Message: message}
}

func invoke(userData unsafe.Pointer, ctx *C.kt_algorithm_context_t, method string) (out NextStep) {
	defer func() {
		if recovered := recover(); recovered != nil {
			out = Fatal
		}
	}()
	state := cgo.Handle(C.kt_go_handle_value(userData)).Value().(*runtimeState)
	goCtx := &Context{ptr: ctx}
	switch method {
	case "setup":
		return state.node.Setup(goCtx)
	case "step":
		return state.node.Step(goCtx)
	case "close":
		return state.node.Close(goCtx)
	default:
		return Fatal
	}
}

//export goKtSetup
func goKtSetup(userData unsafe.Pointer, ctx *C.kt_algorithm_context_t) C.kt_algorithm_outcome_t {
	return C.kt_algorithm_outcome_t(invoke(userData, ctx, "setup"))
}

//export goKtStep
func goKtStep(userData unsafe.Pointer, ctx *C.kt_algorithm_context_t) C.kt_algorithm_outcome_t {
	return C.kt_algorithm_outcome_t(invoke(userData, ctx, "step"))
}

//export goKtClose
func goKtClose(userData unsafe.Pointer, ctx *C.kt_algorithm_context_t) C.kt_algorithm_outcome_t {
	return C.kt_algorithm_outcome_t(invoke(userData, ctx, "close"))
}
