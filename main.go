package main

/*
#include "httpd_cb.h"
#include <string.h>
#include <stdio.h>
static fn_client_accepted client_accepted = NULL;

static void set_httpd_cb(void* cb) {
	client_accepted = (fn_client_accepted)cb;
}

static void request_coming(int client_id) {
	client_accepted(client_id);
}

static void iter_env(void* iter_cb, void* udd, char* key, int keyLen, char* val, int valLen) {
	fn_iter_env iter_env_cb = (fn_iter_env)iter_cb;
	iter_env_cb(udd, key, keyLen, val, valLen);
}
*/
import "C"

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
	"fmt"
	"os"
	"net"
	"unsafe"
	"strings"
	"reflect"
)

type Client struct {
	w http.ResponseWriter
	r *http.Request
	status int
	bytesSent int

	jsonBody map[string]interface{}
	bodyConsumed bool
}

type ListenParam struct {
	host string
	port int32
}

var (
	server_started bool
	listener net.Listener
	starter_req chan ListenParam
	starter_resp chan error
	show_log bool
)

func serve_http(w http.ResponseWriter, r *http.Request) {
	// fmt.Printf("client accepted in Golang\n")
	var startTime, endTime time.Time
	if show_log {
		startTime = time.Now()
	}
	client := &Client{w:w, r:r, status:http.StatusOK}

	clientId := NewObjId(client)
	defer FreeObjId(clientId)

	C.request_coming(C.int(clientId))

	if !show_log {
		return
	}
	endTime = time.Now()
	// 127.0.0.1 - - [06/Oct/2018 16:28:23] "POST / HTTP/1.1" 200 8
	// Mon Jan 2 15:04:05 -0700 MST 2006
	duration := endTime.Sub(startTime)
	fmt.Fprintf(os.Stderr, "%s - - [%s] %v \"%s %s %s\" %d %d\n",
		r.RemoteAddr,
		startTime.Format("2/Jan/2006 15:04:05 -0700 MST"),
		duration,
		r.Method,
		r.RequestURI,
		r.Proto,
		client.status,
		client.bytesSent,
	)
}

// goroutine started by init()
func listener_starter() {
	// fmt.Printf("listener_starter is running\n")
	http.HandleFunc("/", serve_http)

	for {
		param := <-starter_req
		if server_started {
			continue
		}

		var e error
		if strings.HasPrefix(param.host, "unix:") {
			fn := param.host[5:]
			listener, e = net.Listen("unix", fn)
			fmt.Fprintf(os.Stderr, "I am listening at %s\n", param.host)
		} else {
			server := fmt.Sprintf("%s:%d", param.host, param.port)
			listener, e = net.Listen("tcp", server)
			fmt.Fprintf(os.Stderr, "I am listening at %s\n", server)
		}
		if e != nil {
			starter_resp <- e
			continue
		}

		server_started = true
		starter_resp <- nil

		err := http.Serve(listener, nil)
		if err != nil {
			server_started = false
			fmt.Fprintf(os.Stderr, "I was closed: %v\n", err)
		}
	}
}

func init() {
	starter_req = make(chan ListenParam)
	starter_resp = make(chan error)

	go listener_starter()
}

// to construct C NULL
var zero uint64 = uint64(0)
func null() *C.char {
	return (*C.char)(unsafe.Pointer(uintptr(zero)))
}

//export StartHttpd
func StartHttpd(host *C.char, port C.int, httpd_cb unsafe.Pointer, showLog C.int) C.int {
	if server_started {
		fmt.Fprintf(os.Stderr, "httpd server already started\n")
		return C.int(-1)
	}
	show_log = (showLog != 0)
	param := ListenParam{C.GoString(host), int32(port)}
	starter_req <- param
	err := <-starter_resp
	if err != nil {
		fmt.Println(err.Error())
		return C.int(-2)
	}
	C.set_httpd_cb(httpd_cb);
	return C.int(0)
}

//export StopHttpd
func StopHttpd() {
	if server_started {
		listener.Close()
		server_started = false
	}
}

func set_env(goVal *string, val **C.char, valLen *C.int) {
	v := (*reflect.StringHeader)(unsafe.Pointer(goVal))
	*val = (*C.char)(unsafe.Pointer(v.Data))
	*valLen = C.int(v.Len)
}

//export GetReqEnv
func GetReqEnv(clientId C.int, name *C.char, val **C.char, valLen *C.int) C.int {
	c := GetObjById(int32(clientId))
	if c == nil {
		return C.int(-1)
	}

	client := c.(*Client)
	r := client.r

	n := C.GoString(name)
	var goVal *string
	switch n {
	case "PATH_INFO":
		goVal = &(r.URL.Path)
	case "QUERY_STRING":
		goVal = &(r.URL.RawQuery)
	case "REQUEST_METHOD":
		goVal = &(r.Method)
	case "SERVER_PROTOCOL":
		goVal = &(r.Proto)
	case "REMOTE_ADDR":
		goVal = &(r.RemoteAddr)
	case "REQUEST_URI":
		goVal = &(r.RequestURI)
	default:
		vv, ok := r.Header[n]
		if !ok {
			return C.int(-2)
		}
		goVal = &(vv[0])
	}

	set_env(goVal, val, valLen)
	return C.int(0)
}

//export GetJSONVal
func GetJSONVal(clientId C.int, name *C.char, val **C.char, valLen *C.int) C.int {
	c := GetObjById(int32(clientId))
	if c == nil {
		return C.int(-1)
	}

	client := c.(*Client)
	if !client.bodyConsumed {
		fmt.Fprintf(os.Stderr, "please call ReadJSON first\n")
		return C.int(-2)
	}
	if client.jsonBody == nil {
		fmt.Fprintf(os.Stderr, "no JSON read\n")
		return C.int(-3)
	}

	n := C.GoString(name)
	v, ok := client.jsonBody[n]
	if !ok {
		return C.int(-2)
	}
	if v == nil {
		*val = null();
		*valLen = C.int(0)
		return C.int(0)
	}
	switch vv := v.(type) {
	case string:
		set_env(&vv, val, valLen)
		return C.int(0)
	case float64:
		if float64(int64(vv)) == vv {
			d := fmt.Sprintf("%d", int64(vv))
			set_env(&d, val, valLen)
		} else {
			f := fmt.Sprintf("%f", vv)
			set_env(&f, val, valLen)
		}
		return C.int(0)
	case bool:
		b := fmt.Sprintf("%v", vv)
		set_env(&b, val, valLen)
		return C.int(0)
	default:
		return C.int(-3)
	}
}

//export GetFormVal
func GetFormVal(clientId C.int, name *C.char, val **C.char, valLen *C.int) C.int {
	c := GetObjById(int32(clientId))
	if c == nil {
		return C.int(-1)
	}

	client := c.(*Client)
	r := client.r

	n := C.GoString(name)
	v := r.FormValue(n)
	if len(v) == 0 {
		*val = null()
		*valLen = 0
	} else {
		set_env(&v, val, valLen)
	}

	return C.int(0)
}

func iter_env(iter_cb unsafe.Pointer, udd unsafe.Pointer, key, val string) {
	k := (*reflect.StringHeader)(unsafe.Pointer(&key))
	v := (*reflect.StringHeader)(unsafe.Pointer(&val))

	C.iter_env(iter_cb, udd,
	           (*C.char)(unsafe.Pointer(k.Data)), C.int(k.Len),
	           (*C.char)(unsafe.Pointer(v.Data)), C.int(v.Len))
}

//export IterReqEnvs
func IterReqEnvs(clientId C.int, iter_cb unsafe.Pointer, udd unsafe.Pointer) {
	c := GetObjById(int32(clientId))
	if c == nil {
		return
	}

	client := c.(*Client)
	r := client.r

	iter_env(iter_cb, udd, "PATH_INFO", r.URL.Path)
	iter_env(iter_cb, udd, "QUERY_STRING", r.URL.RawQuery)
	iter_env(iter_cb, udd, "REQUEST_METHOD", r.Method)
	iter_env(iter_cb, udd, "SERVER_PROTOCOL", r.Proto)
	iter_env(iter_cb, udd, "REMOTE_ADDR", r.RemoteAddr)

	for k, vs := range r.Header {
		for _, v := range vs {
			iter_env(iter_cb, udd, k, v);
		}
	}
}

func set_body(goBody []byte, body **C.char, bodyLen *C.int) {
	p := (*reflect.SliceHeader)(unsafe.Pointer(&goBody))
	*body = (*C.char)(unsafe.Pointer(p.Data))
	*bodyLen = C.int(p.Len)
}

//export ReadBody
func ReadBody(clientId C.int, body **C.char, bodyLen *C.int) C.int {
	c := GetObjById(int32(clientId))
	if c == nil {
		return C.int(-1)
	}

	client := c.(*Client)
	r := client.r
	switch {
	case r.Method == "" || r.Method == "GET" || r.Method == "HEAD" || r.ContentLength == 0:
		*body = null();
		*bodyLen = C.int(0)
		return C.int(0)
	case r.ContentLength < 0:
		return C.int(http.StatusLengthRequired)
	case r.Body == nil:
		return C.int(http.StatusBadRequest)
	default:
		if client.bodyConsumed {
			return C.int(-2)
		}
		defer func() {
			client.bodyConsumed = true
		}()
		b, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return C.int(http.StatusInternalServerError)
		}
		set_body(b, body, bodyLen)
		return C.int(0)
	}
}

//export ReadJSON
func ReadJSON(clientId C.int) C.int {
	c := GetObjById(int32(clientId))
	if c == nil {
		return C.int(-1)
	}

	client := c.(*Client)
	r := client.r
	if r.Body == nil {
		return C.int(http.StatusBadRequest)
	}

	if client.bodyConsumed {
		return C.int(-2)
	}
	defer func() {
		client.bodyConsumed = true
	}()
	if err := json.NewDecoder(r.Body).Decode(&client.jsonBody); err != nil {
		fmt.Fprintf(os.Stderr, "failed to to ReadJSON: %v\n", err)
		return C.int(-3)
	}
	return C.int(0)
}

//export SetStatus
func SetStatus(clientId C.int, code C.int) C.int {
	c := GetObjById(int32(clientId))
	if c == nil {
		return C.int(-1)
	}

	client := c.(*Client)
	w := client.w

	status := int(code)
	w.WriteHeader(status)
	client.status = status
	return C.int(0)
}

//export SetRespHeader
func SetRespHeader(clientId C.int, name, val *C.char) C.int {
	c := GetObjById(int32(clientId))
	if c == nil {
		return C.int(-1)
	}

	client := c.(*Client)
	w := client.w
	w.Header().Set(C.GoString(name), C.GoString(val))
	return C.int(0)
}

//export AddRespHeader
func AddRespHeader(clientId C.int, name, val *C.char) C.int {
	c := GetObjById(int32(clientId))
	if c == nil {
		return C.int(-1)
	}

	client := c.(*Client)
	w := client.w
	w.Header().Add(C.GoString(name), C.GoString(val))
	return C.int(0)
}

//export OutputChunk
func OutputChunk(clientId C.int, chunk *C.char, length C.int) C.int {
	c := GetObjById(int32(clientId))
	if c == nil {
		return C.int(-1)
	}

	client := c.(*Client)
	w := client.w

	if length < 0 {
		length = C.int(C.strlen(chunk))
	}

	// construct []byte with the same memory of chunk
	var b []byte
	var bs = (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bs.Data = uintptr(unsafe.Pointer(chunk))
	bs.Len = int(length)
	bs.Cap = int(length)

	bytesSent, err := w.Write(b)
	if err != nil {
		return C.int(-1)
	}
	client.bytesSent += bytesSent
	return C.int(bytesSent)
}

//export OutputJSONError
func OutputJSONError(clientId C.int, code C.int, msg *C.char) C.int {
	c := GetObjById(int32(clientId))
	if c == nil {
		return C.int(-1)
	}

	client := c.(*Client)
	w := client.w

	w.Header().Set("Content-Type", "application/json")
	status := int(code)
	w.WriteHeader(status)
	b, err := json.Marshal(map[string]interface{}{
		"code": status,
		"msg": C.GoString(msg),
	})
	if err != nil {
		return C.int(-2)
	}
	bytesSent, err := w.Write(b)
	if err != nil {
		return C.int(-3)
	}
	client.bytesSent += bytesSent
	return C.int(bytesSent)
}

//export HttpdLoop
func HttpdLoop() {
	c := make(chan struct{})
	<-c
}

func main() {
}
