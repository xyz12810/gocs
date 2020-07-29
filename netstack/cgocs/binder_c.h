typedef void (*pack_writer)(void *buffer, int length);
typedef void *(*conn_dialer)(void *remote, int length);
typedef int (*conn_writer)(void *conn, void *buffer, int length);
typedef void (*conn_closer)(void *conn);
typedef void (*log_writer)(void *buffer, int length);

void cgo_pack_write(void *buffer, int length);
void *cgo_conn_dial(void *remote, int length);
int cgo_conn_write(void *conn, void *buffer, int length);
void cgo_conn_close(void *conn);
void cgo_log_write(void *buffer, int length);

extern pack_writer cs_pack_writer;
extern conn_dialer cs_conn_dialer;
extern conn_writer cs_conn_writer;
extern conn_closer cs_conn_closer;
extern log_writer cs_log_writer;