package coversocks

//
//typedef void (*pack_writer)(void* buffer,int length);
//typedef long (*conn_dialer)(void* remote,int length);
//typedef int (*conn_writer)(long cid,void* buffer,int length);
//typedef void (*conn_closer)(long cid);
//pack_writer cs_pack_writer=0;
//conn_dialer cs_conn_dialer=0;
//conn_writer cs_conn_writer=0;
//conn_closer cs_conn_closer=0;
//void cgo_pack_write(void* buffer,int length){if(cs_pack_writer)cs_pack_writer(buffer,length);}
//long cgo_conn_dial(void* remote,int length){if(cs_conn_dialer)return cs_conn_dialer(remote,length);else return 0;}
//int cgo_conn_write(long cid,void* buffer,int length){if(cs_conn_writer)return cs_conn_writer(cid,buffer,length);else return -1;}
//void cgo_conn_close(long cid){if(cs_conn_closer)cs_conn_closer(cid);}
import "C"
import "unsafe"

func cgoPackWrite(buffer []byte, length int) {
	C.cgo_pack_write(unsafe.Pointer(&buffer[0]), C.int(length))
}

func cgoConnDial(buffer []byte, length int) int64 {
	return int64(C.cgo_conn_dial(unsafe.Pointer(&buffer[0]), C.int(length)))
}

func cgoConnWrite(cid int64, buffer []byte, length int) int {
	return int(C.cgo_conn_write(C.long(cid), unsafe.Pointer(&buffer[0]), C.int(length)))
}

func cgoConnClose(cid int64) {
	C.cgo_conn_close(C.long(cid))
}
