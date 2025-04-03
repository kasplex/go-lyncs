
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
	"_G": {"jit","collectgarbage","rawget","rawset","loadfile","load","loadstring","dofile","gcinfo","coroutine","debug","getfenv"/*,"print"*/},
}

var stateReadonlyList = `
	table = _set(table)
	string = _set(string)
	math = _set(math)
	bit = _set(bit)
	-- built-in ...
`

////////////////////////////////
func stateFromCode(code string) (*C.lua_State, []byte, error) {
	s := C.luaL_newstate()
	if s == nil {
		return nil, nil, fmt.Errorf("creation failed @stateFromCode")
	}
	codeSandbox, err := stateSandbox(s)
	if err != nil {
		stateClose(s)
		return nil, nil, err
	}
	cCode := C.CString(codeSandbox + code)
	defer C.free(unsafe.Pointer(cCode))
	if C.LUA_OK != C.luaL_loadstring(s, cCode) {
		stateClose(s)
		return nil, nil, fmt.Errorf("load failed @stateFromCode")
	}
	var bc []byte
	var buffer C.bcBuffer
	n := C.bcDump(s, &buffer)
	if n > 0 {
		bc = C.GoBytes(unsafe.Pointer(buffer.bc), C.int(n))
		defer C.free(unsafe.Pointer(buffer.bc))
	}
	err = stateCall(s)
	if err != nil {
		stateClose(s)
		return nil, nil, err
	}

	// check sc-func exists ...

	return s, bc, nil
}

////////////////////////////////
func stateFromBC(bc []byte) (*C.lua_State, error) {
	s := C.luaL_newstate()
	if s == nil {
		return nil, fmt.Errorf("creation failed @stateFromBC")
	}
	_, err := stateSandbox(s)
	if err != nil {
		stateClose(s)
		return nil, err
	}
	if C.LUA_OK != C.luaL_loadbuffer(s, (*C.char)(unsafe.Pointer(&bc[0])), C.size_t(len(bc)), (*C.char)(unsafe.Pointer(nil))) {
		stateClose(s)
		return nil, fmt.Errorf("load failed @stateFromBC")
	}
	err = stateCall(s)
	if err != nil {
		stateClose(s)
		return nil, err
	}

	// check sc-func exists ...

	return s, nil
}

////////////////////////////////
func stateSandbox(s *C.lua_State) (string, error) {
	var err error
	C.luaopen_base(s)
	C.luaopen_math(s)
	C.luaopen_string(s)
	C.luaopen_table(s)
	C.luaopen_bit(s)
	C.luaopen_jit(s)
	C.luaopen_string_buffer(s)
	for k, v := range stateRemoveMap {
		err = stateSetTableFieldNil(s, k, v)
		if err != nil {
			return "", err
		}
	}
	stateSetTableFieldString(s, "_G", []string{"_VERSION"}, []string{"Lua 5.1 Lyncs"})
	codeSandbox := ""

	// builtin ...

	codeSandbox += `
function setRO()
	local _set = function (t)
		local p = {}
		local mt = {
			__index = t,
			__newindex = function(_, k, v)
				if t ~= _G then
					error("variable read-only", 2)
				end
				if k=="session" then
					t[k] = v
					return
				end
				if (k=="init" or k=="run") and t[k]==nil and type(v)=="function" then
					t[k] = v
					return
				end
				error("variable read-only", 2)
			end
		}
		setmetatable(p, mt)
		return p
	end
` + stateReadonlyList + `
	setfenv(2, _set(_G))
	setfenv = nil
	setRO = nil
end
setRO();
`
	return codeSandbox, nil
}

////////////////////////////////
func stateCall(s *C.lua_State) (error) {
	if C.LUA_OK != C.lua_pcall(s, 0, 0, 0) {
		msg := C.GoString(C.lua_tolstring(s, -1, nil))
		C.lua_settop(s, C.lua_gettop(s)-1)
		msgS := strings.Split(msg, `"]:`)
		if len(msgS) < 2 {
			return fmt.Errorf(msg + " @stateCall")
		}
		if len(msgS[1]) > 30 {
			msgS[1] = msgS[1][:30] + ".."
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

