#ifndef _GO_HTTPD_CB_H_
#define _GO_HTTPD_CB_H_

#ifdef __cplusplus
extern "C" {
#endif

typedef void (*fn_client_accepted)(int client_id);
typedef void (*fn_iter_env)(void *udd, char* key, int key_len, char* val, int val_len);

#ifdef __cplusplus
}
#endif

#endif
