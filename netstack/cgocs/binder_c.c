#include "binder_c.h"

pack_writer cs_pack_writer = 0;
conn_dialer cs_conn_dialer = 0;
conn_writer cs_conn_writer = 0;
conn_closer cs_conn_closer = 0;
log_writer cs_log_writer = 0;

void cgo_pack_write(void *buffer, int length) {
  if (cs_pack_writer)
    cs_pack_writer(buffer, length);
}
void *cgo_conn_dial(void *remote, int length) {
  if (cs_conn_dialer)
    return cs_conn_dialer(remote, length);
  else
    return 0;
}
int cgo_conn_write(void *conn, void *buffer, int length) {
  if (cs_conn_writer)
    return cs_conn_writer(conn, buffer, length);
  else
    return -1;
}
void cgo_conn_close(void *conn) {
  if (cs_conn_closer)
    cs_conn_closer(conn);
}

void cgo_log_write(void *buffer, int length) {
  if (cs_log_writer)
    cs_log_writer(buffer, length);
}