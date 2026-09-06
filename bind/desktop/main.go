package main

/*
#include <stdlib.h>
#include <stdint.h>

typedef void (*TaskEventCallback)(char* payload);

typedef void (*InvokeResultCallback)(uint64_t request_id, int success, char* payload);

static void callTaskEventCallback(uintptr_t callback, char* payload) {
	((TaskEventCallback)callback)(payload);
}

static void callInvokeResultCallback(uintptr_t callback, uint64_t request_id, int success, char* payload) {
	((InvokeResultCallback)callback)(request_id, success, payload);
}
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"

	nativebridge "github.com/GopeedLab/gopeed/bind/native"
	"github.com/GopeedLab/gopeed/pkg/rest"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

func main() {}

//export Start
func Start(cfg *C.char) (int, *C.char) {
	var config model.StartConfig
	if err := json.Unmarshal([]byte(C.GoString(cfg)), &config); err != nil {
		return 0, C.CString(err.Error())
	}
	config.ProductionMode = true
	config.NativeMode = true
	applyWebViewProvider(&config)
	realPort, err := rest.Start(&config)
	if err != nil {
		return 0, C.CString(err.Error())
	}
	nativebridge.ResumeInvokes()
	return realPort, nil
}

//export Stop
func Stop() {
	nativebridge.Stop()
}

//export GetAPIServerState
func GetAPIServerState() *C.char {
	return C.CString(apiServerResult(rest.GetAPIServerState()))
}

//export StartAPIServer
func StartAPIServer() *C.char {
	return C.CString(apiServerResult(rest.StartAPIServer()))
}

//export StopAPIServer
func StopAPIServer() *C.char {
	return C.CString(apiServerResult(rest.StopAPIServer()))
}

//export RestartAPIServer
func RestartAPIServer() *C.char {
	return C.CString(apiServerResult(rest.RestartAPIServer()))
}

func apiServerResult(state *model.APIServerState, err error) string {
	result := &model.APIServerOperationResult{State: state}
	if err != nil {
		result.Error = err.Error()
	}
	data, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return fmt.Sprintf(`{"state":null,"error":%q}`, marshalErr.Error())
	}
	return string(data)
}

//export InvokeAsync
func InvokeAsync(
	method *C.char,
	path *C.char,
	query *C.char,
	body *C.char,
	requestID C.ulonglong,
	callback C.uintptr_t,
) {
	if callback == 0 {
		return
	}
	complete := func(result string, err error) {
		success := C.int(1)
		payload := result
		if err != nil {
			success = 0
			payload = err.Error()
		}
		C.callInvokeResultCallback(
			callback,
			C.uint64_t(requestID),
			success,
			C.CString(payload),
		)
	}
	nativebridge.InvokeAsync(
		C.GoString(method),
		C.GoString(path),
		C.GoString(query),
		C.GoString(body),
		complete,
	)
}

//export SubscribeTaskEvents
func SubscribeTaskEvents(mask C.ulonglong, callback C.uintptr_t) {
	if mask == 0 || callback == 0 {
		rest.SubscribeTaskEvents(0, nil)
		return
	}
	rest.SubscribeTaskEvents(uint64(mask), func(payload string) {
		// Ownership crosses the asynchronous Dart callback boundary. Dart frees
		// this string with FreeCString after decoding it.
		C.callTaskEventCallback(callback, C.CString(payload))
	})
}

//export FreeCString
func FreeCString(value *C.char) {
	if value != nil {
		C.free(unsafe.Pointer(value))
	}
}
