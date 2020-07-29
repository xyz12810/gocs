/* Code generated by cmd/cgo; DO NOT EDIT. */

/* package github.com/coversocks/gocs/netstack/cgocs */


#line 1 "cgo-builtin-export-prolog"

#include <stddef.h> /* for ptrdiff_t below */

#ifndef GO_CGO_EXPORT_PROLOGUE_H
#define GO_CGO_EXPORT_PROLOGUE_H

#ifndef GO_CGO_GOSTRING_TYPEDEF
typedef struct { const char *p; ptrdiff_t n; } _GoString_;
#endif

#endif

/* Start of preamble from import "C" comments.  */





/* End of preamble from import "C" comments.  */


/* Start of boilerplate cgo prologue.  */
#line 1 "cgo-gcc-export-header-prolog"

#ifndef GO_CGO_PROLOGUE_H
#define GO_CGO_PROLOGUE_H

typedef signed char GoInt8;
typedef unsigned char GoUint8;
typedef short GoInt16;
typedef unsigned short GoUint16;
typedef int GoInt32;
typedef unsigned int GoUint32;
typedef long long GoInt64;
typedef unsigned long long GoUint64;
typedef GoInt64 GoInt;
typedef GoUint64 GoUint;
typedef __SIZE_TYPE__ GoUintptr;
typedef float GoFloat32;
typedef double GoFloat64;
typedef float _Complex GoComplex64;
typedef double _Complex GoComplex128;

/*
  static assertion to make sure the file is being used on architecture
  at least with matching size of GoInt.
*/
typedef char _check_for_64_bit_pointer_matching_GoInt[sizeof(void*)==64/8 ? 1:-1];

#ifndef GO_CGO_GOSTRING_TYPEDEF
typedef _GoString_ GoString;
#endif
typedef void *GoMap;
typedef void *GoChan;
typedef struct { void *t; void *v; } GoInterface;
typedef struct { void *data; GoInt len; GoInt cap; } GoSlice;

#endif

/* End of boilerplate cgo prologue.  */

#ifdef __cplusplus
extern "C" {
#endif


//cgo_cs_bootstrap will bootstrap by config file path

extern GoString cs_bootstrap(GoString p0, GoInt p1, GoUint8 p2, GoString p3);

//cs_inbound_write write inbound data to the netstack

extern GoInt cs_inbound_write(GoSlice p0, GoInt p1, GoInt p2);

//cs_dial_done write conn data to the netstack

extern GoInt cs_dial_done(void* p0, GoInt p1);

//cs_conn_write write conn data to the netstack

extern GoInt cs_conn_write(void* p0, GoSlice p1, GoInt p2, GoInt p3);

//cs_conn_close close the connection

extern void cs_conn_close(void* p0);

extern void cs_hello();

//cs_start the process

extern GoString cs_start();

//stop the process

extern GoString cs_stop();

//cs_proxy_set will add proxy setting by key

extern GoString cs_proxy_set(GoString p0, GoUint8 p1);

//cs_change_mode will change proxy mode by global/auto

extern GoString cs_change_mode(GoString p0);

//cs_proxy_mode will return current proxy mode

extern GoString cs_proxy_mode();

//cs_test_web will test web request

extern void cs_test_web(GoString p0, GoString p1);

#ifdef __cplusplus
}
#endif
