
////////////////////////////////
package lyncs

/*
#include <stdlib.h>
#include "lua.h"
#include "lualib.h"
#include "lauxlib.h"
#include "./bridge.h"
*/
import "C"
import (
    "fmt"
    "unsafe"
    "strings"
    
    // ...
    
)

////////////////////////////////
var stateRemoveMap = map[string][]string{
    "string": {"dump"},
    "math": {"randomseed","random"},
    "_G": {"jit","collectgarbage","getfenv","setfenv","rawget","rawset","loadfile","load","loadstring","dofile","gcinfo","coroutine","debug"/*,"print"*/},
}

var stateReadonlyList = []string{"_G","table","string","math","bit"/*,built-in..*/}

////////////////////////////////
func stateFromCode(code string) (*C.lua_State, error) {
    s := C.luaL_newstate()
    if s == nil {
        return nil, fmt.Errorf("creation failed @stateFromCode")
    }
    stateLibOpen(s)
    cCode := C.CString(code)
    defer C.free(unsafe.Pointer(cCode))
    if C.LUA_OK != C.luaL_loadstring(s, cCode) {
        stateClose(s)
        return nil, fmt.Errorf("load failed @stateFromCode")
    }
    err := stateLibFilter(s)
    if err != nil {
        stateClose(s)
        return nil, err
    }
    
    // builtin ...
    
    stateSetGlobalReadonly(s)
    err = stateCall(s)
    if err != nil {
        stateClose(s)
        return nil, err
    }
    
    // check sc-func exists ...
    // dump ...
    
    return s, nil
}

////////////////////////////////
func stateFromBuffer(buffer []byte) (*C.lua_State, error) {
    s := C.luaL_newstate()
    if s == nil {
        return nil, fmt.Errorf("creation failed @stateFromBuffer")
    }
    stateLibOpen(s)
    
    // load buffer ...

    err := stateLibFilter(s)
    if err != nil {
        stateClose(s)
        return nil, err
    }

    // builtin ...
    
    stateSetGlobalReadonly(s)
    
    // pcall ...
    // check sc-func exists ...
    
    return s, nil
}

////////////////////////////////
func stateLibOpen(s *C.lua_State) {
    C.luaopen_base(s)
    C.luaopen_math(s)
    C.luaopen_string(s)
    C.luaopen_table(s)
    C.luaopen_bit(s)
    C.luaopen_jit(s)
    C.luaopen_string_buffer(s)
}

////////////////////////////////
func stateLibFilter(s *C.lua_State) (error) {
    var err error
    for k, v := range stateRemoveMap {
        err = stateSetTableFieldNil(s, k, v)
        if err != nil {
            return err
        }
    }
    stateSetTableFieldString(s, "_G", []string{"_VERSION"}, []string{"Lua 5.1 Lyncs"})
    return nil
}

////////////////////////////////
func stateCall(s *C.lua_State) (error) {
    if C.LUA_OK != C.lua_pcall(s, 0, 0, 0) {
        msg := C.GoString(C.lua_tolstring(s, -1, nil))
        C.lua_settop(s, C.lua_gettop(s)-1)
        msgS := strings.Split(msg, `"]:`)
        if len(msgS[1]) > 40 {
            msgS[1] = msgS[1][:40] + ".."
        }
        return fmt.Errorf(msgS[1] + " @stateCall")
    }
    return nil
}

////////////////////////////////
func stateClose(s *C.lua_State) {
    C.lua_close(s)
}

////////////////////////////////
func stateClean(s *C.lua_State) {

    // ...
    
}

////////////////////////////////
func stateSetTableReadonly(s *C.lua_State, table string) {
    ct := C.CString(table)
    defer C.free(unsafe.Pointer(ct))
    ctRo := C.CString(table+"_ro")
    defer C.free(unsafe.Pointer(ctRo))
    cf := C.CString("__newindex")
    defer C.free(unsafe.Pointer(cf))
    C.luaL_newmetatable(s, ctRo)
    C.lua_pushstring(s, cf)
    if table == "_G" {
        C.lua_pushcclosure(s, C.lua_CFunction(C._g_mt_newindex), 0)
    } else {
        C.lua_pushcclosure(s, C.lua_CFunction(C._mt_newindex), 0)
    }
    C.lua_settable(s, -3)
    C.lua_getfield(s, C.LUA_GLOBALSINDEX, ct)
    C.lua_pushvalue(s, -2)
    C.lua_setmetatable(s, -2)
    C.lua_settop(s, C.lua_gettop(s)-2)
}

////////////////////////////////
func stateSetGlobalReadonly(s *C.lua_State) {
    for _, t := range stateReadonlyList {
        stateSetTableReadonly(s, t)
    }
}

////////////////////////////////
func stateSetTableField(s *C.lua_State, table string, field []string, fSet func(*C.lua_State, int)) (error) {
    ct := C.CString(table)
    defer C.free(unsafe.Pointer(ct))
    C.lua_getfield(s, C.LUA_GLOBALSINDEX, ct)
    if C.lua_type(s, 1) == C.LUA_TTABLE {
        return fmt.Errorf("not a table @stateSetTableField")
    }
    for i, f := range field {
        cf := C.CString(f)
        defer C.free(unsafe.Pointer(cf))
        C.lua_pushstring(s, cf)
        fSet(s, i)
        C.lua_settable(s, -3)
    }
    C.lua_settop(s, C.lua_gettop(s)-1)
    return nil
}

////////////////////////////////
func stateSetTableFieldNil(s *C.lua_State, table string, field []string) (error) {
    return stateSetTableField(s, table, field, func(s *C.lua_State, i int) {
        C.lua_pushnil(s)
    })
}

////////////////////////////////
func stateSetTableFieldString(s *C.lua_State, table string, field []string, value []string) (error) {
    return stateSetTableField(s, table, field, func(s *C.lua_State, i int) {
        cv := C.CString(value[i])
        defer C.free(unsafe.Pointer(cv))
        C.lua_pushstring(s, cv)
    })
}

// ...

