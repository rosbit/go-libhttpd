/*
 * used to generate a sample using go-libhttpd
 * gcc -o sample sample.c -I.. -L.. -lhttpd
 */
#include <stdio.h>
#include "httpd_cb.h"
#include "go_httpd.h"

static const char* env_names[] = {
	"PATH_INFO",
	"QUERY_STRING",
	"REQUEST_METHOD",
	"SERVER_PROTOCOL",
	"REMOTE_ADDR",
	"Content-Length",
	"User-Agent",
	NULL
};

static void iter_envs(void* udd, char* key, int key_len, char* val, int val_len) {
	printf("env: %.*s => %.*s\n", key_len, key, val_len, val);
}

static void client_accepted(int client_id) {
	const char* name;
	char* val;
	int val_len;
	int i;
	int res;

	// Get Request
	for (i=0; env_names[i] != NULL; ++i) {
		name = env_names[i];
		res = GetReqEnv(client_id, (char*)name, &val, &val_len);
		if (res == 0) {
			printf("env: %s => \"%.*s\"\n", name, val_len, val);
			continue;
		}
		printf("no env %s found\n", name);
	}
	// Iter Env
	IterReqEnvs(client_id, iter_envs, NULL);

	// Read Body
	res = ReadBody(client_id, &val, &val_len);
	if (res != 0) {
		printf("errCode when reading body: %d\n", res);
	} else {
		printf("body: %.*s\n", val_len, val);
	}

	// send RESPONSE
	/*
	 * step 1: call SetRespHeader() for each header
	 * step 2: call SetStatus(). All the headers will be sent to the client.
	 * step 3: call OutputChunk() for each chunk.
	 */
	SetRespHeader(client_id, "X-gohttpd-Header", "gohttpd-value");
	AddRespHeader(client_id, "X-gohttpd-Header", "gohttpd-value-2");
	SetStatus(client_id, 200);
	OutputChunk(client_id, "this is my result\n", -1);
	OutputChunk(client_id, "this is also my result\n", -1);
}

int start_server(char *host, int port, int show_log) {
	return StartHttpd(host, port, client_accepted, show_log);
}

void stop_server() {
	StopHttpd();
}

int main() {
	int ret = start_server("", 8801, 1);
	if (ret != 0) {
		printf("failed to start_server: %d\n", ret);
		return -1;
	}
	getchar();
	return 0;
}
