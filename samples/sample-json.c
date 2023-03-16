/*
 * used to generate a sample using go-libhttpd
 * gcc -o sample-json sample-json.c -I.. -L.. -lhttpd
 */
#include <stdio.h>
#include "httpd_cb.h"
#include "go_httpd.h"

static void client_accepted(int client_id) {
	const char* name;
	char* val;
	int val_len;
	int res;
	char strRes[256];

	SetRespHeader(client_id, "Content-Type", "text/plain");

	// ReadBody
	res = ReadJSON(client_id);
	if (res != 0) {
		sprintf(strRes, "failed to ReadJSON: %d\n", res);
		OutputChunk(client_id, strRes, -1);
		return;
	}

	// Get Value from JSON
	res = GetJSONVal(client_id, "name", &val, &val_len);
	if (res != 0) {
		sprintf(strRes, "failed to GetJSONVal: %d\n", res);
		OutputChunk(client_id, strRes, -1);
		return;
	}
	// send RESPONSE
	OutputChunk(client_id, val, val_len);
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
