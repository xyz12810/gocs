package main

//
//#include "binder_c.h"
import "C"
import "unsafe"

func cgoPackWrite(buffer []byte, length int) {
	C.cgo_pack_write(unsafe.Pointer(&buffer[0]), C.int(length))
}

func cgoConnDial(buffer []byte, length int) unsafe.Pointer {
	return C.cgo_conn_dial(unsafe.Pointer(&buffer[0]), C.int(length))
}

func cgoConnWrite(conn unsafe.Pointer, buffer []byte, length int) int {
	return int(C.cgo_conn_write(conn, unsafe.Pointer(&buffer[0]), C.int(length)))
}

func cgoConnClose(conn unsafe.Pointer) {
	C.cgo_conn_close(conn)
}

func cgoLogWrite(buffer []byte, length int) {
	C.cgo_log_write(unsafe.Pointer(&buffer[0]), C.int(length))
}
