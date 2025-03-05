#include <string.h>

////////////////////////////////
__attribute__((weak))
int _g_mt_newindex(lua_State *s) {
    const char *key = lua_tostring(s, 2);
    const char *allowed[] = {"session", "init", "regst", "run", NULL};
    for (int i=0; allowed[i]!=NULL; i++) {
        if (strcmp(key,allowed[i])==0) {
            lua_rawset(s, 1);
            return 0;
        }
    }
    luaL_error(s, "variable read-only");
    return 0;
}

////////////////////////////////
__attribute__((weak))
int _mt_newindex(lua_State *s) {
    luaL_error(s, "variable read-only");
    return 0;
}

// ...
