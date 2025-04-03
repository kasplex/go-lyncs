#include <string.h>

////////////////////////////////
typedef struct {
	char *bc;
	size_t n;
} bcBuffer;
static int bcWriter(lua_State *s, const void *bc, size_t n, void *output) {
	bcBuffer *buffer = (bcBuffer*)output;
	buffer->bc = (char*)realloc(buffer->bc, buffer->n+n);
	memcpy(buffer->bc+buffer->n, bc, n);
	buffer->n += n;
	return 0;
}
static size_t bcDump(lua_State *s, bcBuffer *output) {
	output->bc = NULL;
	output->n = 0;
	if (lua_dump(s, bcWriter, output)!=0) {
		free(output->bc);
		output->bc = NULL;
		output->n = 0;
	}
	return output->n;
}

//__attribute__((weak))
// ...
