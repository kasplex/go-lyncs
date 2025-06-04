
////////////////////////////////
package lyncs

/*
#include <stdlib.h>
#include "lua.h"
#include "lualib.h"
#include "lauxlib.h"
#include "bytecode.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
	"strings"
)

////////////////////////////////
var stateRemoveMap = map[string][]string{
	"string": {"dump"},
	"math": {"randomseed","random"},
	"_G": {"jit","collectgarbage","rawget","rawset","loadfile","load","loadstring","dofile","gcinfo","coroutine","debug","getfenv","pcall","xpcall","newproxy"},
}

var stateReadonlyList = `
	table = _set(table)
	string = _set(string)
	math = _set(math)
	bit = _set(bit)
	mpz = _set(mpz)
`

////////////////////////////////
func stateFromCode(code string) (*C.lua_State, []byte, error) {
	s := C.luaL_newstate()
	if s == nil {
		return nil, nil, fmt.Errorf("creation failed @stateFromCode")
	}
	codeSandbox, err := stateSandbox(s, false)
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
	err = stateCall(s, 0)
	if err != nil {
		stateClose(s)
		return nil, nil, err
	}
	return s, bc, nil
}

////////////////////////////////
func stateFromBC(bc []byte) (*C.lua_State, error) {
	s := C.luaL_newstate()
	if s == nil {
		return nil, fmt.Errorf("creation failed @stateFromBC")
	}
	_, err := stateSandbox(s, true)
	if err != nil {
		stateClose(s)
		return nil, err
	}
	if C.LUA_OK != C.luaL_loadbuffer(s, (*C.char)(unsafe.Pointer(&bc[0])), C.size_t(len(bc)), (*C.char)(unsafe.Pointer(nil))) {
		stateClose(s)
		return nil, fmt.Errorf("load failed @stateFromBC")
	}
	err = stateCall(s, 0)
	if err != nil {
		stateClose(s)
		return nil, err
	}
	return s, nil
}

////////////////////////////////
func stateSandbox(s *C.lua_State, byBC bool) (string, error) {
	var err error
	C.luaopen_base(s)
	C.luaopen_table(s)
	C.luaopen_string(s)
	C.luaopen_string_buffer(s)
	C.luaopen_math(s)
	C.luaopen_bit(s)
	C.luaopen_gmp(s, 256)
	C.luaopen_jit(s)
	C.lua_settop(s, 0)
	for k, v := range stateRemoveMap {
		err = stateSetGlobalTableFieldNil(s, k, v)
		if err != nil {
			return "", err
		}
	}
	stateSetGlobalTableFieldString(s, "_G", []string{"_VERSION"}, []string{"LuaJIT 2.1 Lyncs"})
	if byBC {
		return "", nil
	}
	codeSandbox := ""
	for t, fn := range lRuntime.cfg.Builtin {
		stateReadonlyList += "\t" + t + " = _set("+ t +")\r\n"
		codeSandbox += fn + "\r\n"
	}
	codeCallbacks := "\tlocal fn = {"
	if len(lRuntime.cfg.Callbacks) > 0 {
		for _, v := range lRuntime.cfg.Callbacks {
			codeCallbacks += `["`+v+`"]=true,`
		}
	}
	codeCallbacks += "}"
	codeDebug := "\tprint = function(...) end;"
	if lRuntime.cfg.Debug {
		codeDebug = ""
	}
	codeSandbox += `
function setRO()
` + codeCallbacks + `
	local _set = function (t)
		local _setmt = _G.setmetatable
		if t==_G then _G.setmetatable=nil; _G.getmetatable=nil end
		local p = {}
		local mt = {
			__index = t,
			__newindex = function(_, k, v)
				if t~=_G then error("variable read-only",2) end
				if fn[k] and t[k]==nil and type(v)=="function" then t[k]=v; return end
				error("variable read-only", 2)
			end
		}
		_setmt(p, mt)
		return p
	end
` + codeDebug + `
` + stateReadonlyList + `
	_G.setfenv(2, _set(_G))
	_G.setfenv = nil
	_G.setRO = nil
end
setRO()
local _;
`
	return codeSandbox, nil
}

////////////////////////////////
func stateCall(s *C.lua_State, nResult C.int) (error) {
	if C.LUA_OK != C.lua_pcall(s, 0, nResult, 0) {
		return stateError(s, "stateCall")
	}
	return nil
}

////////////////////////////////
func stateCallFunc(s *C.lua_State, fn string, nResult C.int) (error) {
	cFunc := C.CString(fn)
	defer C.free(unsafe.Pointer(cFunc))
	C.lua_getfield(s, C.LUA_GLOBALSINDEX, cFunc)
	return stateCall(s, nResult)
}

////////////////////////////////
func stateCheckFunc(s *C.lua_State, fn string) (bool) {
	cFunc := C.CString(fn)
	defer C.free(unsafe.Pointer(cFunc))
	C.lua_getfield(s, C.LUA_GLOBALSINDEX, cFunc)
	defer C.lua_settop(s, C.lua_gettop(s)-1)
	if C.lua_type(s, -1) != C.LUA_TFUNCTION {
		return false
	}
	return true
}

////////////////////////////////
func stateError(s *C.lua_State, caller string) (error) {
	msg := C.GoString(C.lua_tolstring(s, -1, nil))
	C.lua_settop(s, C.lua_gettop(s)-1)
	msgS := strings.Split(msg, `"]:`)
	if len(msgS) < 2 {
		return fmt.Errorf(msg + " @"+caller)
	}
	if len(msgS[1]) > 30 {
		msgS[1] = msgS[1][:30] + ".."
	}
	return fmt.Errorf(msgS[1] + " @"+caller)
}

////////////////////////////////
func stateClose(s *C.lua_State) {
	C.lua_close(s)
}

////////////////////////////////
func stateClean(s *C.lua_State) {
	C.lua_settop(s, 0)
	stateSetGlobalTableFieldNil(s, "_G", []string{"session", "state"})
	C.lua_gc(s, C.LUA_GCCOLLECT, 0)
}

////////////////////////////////
func stateSetGlobalTableField(s *C.lua_State, table string, field []string, fSet func(*C.lua_State, int)) (error) {
	ct := C.CString(table)
	defer C.free(unsafe.Pointer(ct))
	C.lua_getfield(s, C.LUA_GLOBALSINDEX, ct)
	defer C.lua_settop(s, C.lua_gettop(s)-1)
	if C.lua_type(s, 1) != C.LUA_TTABLE {
		return fmt.Errorf("not a table @stateSetGlobalTableField")
	}
	for i, fn := range field {
		cf := C.CString(fn)
		defer C.free(unsafe.Pointer(cf))
		C.lua_pushstring(s, cf)
		fSet(s, i)
		C.lua_settable(s, -3)
	}
	return nil
}

////////////////////////////////
func stateSetGlobalTableFieldNil(s *C.lua_State, table string, field []string) (error) {
	return stateSetGlobalTableField(s, table, field, func(s *C.lua_State, i int) {
		C.lua_pushnil(s)
	})
}

////////////////////////////////
func stateSetGlobalTableFieldString(s *C.lua_State, table string, field []string, value []string) (error) {
	return stateSetGlobalTableField(s, table, field, func(s *C.lua_State, i int) {
		cv := C.CString(value[i])
		defer C.free(unsafe.Pointer(cv))
		C.lua_pushstring(s, cv)
	})
}

////////////////////////////////
func stateSetTableByMap1(s *C.lua_State, m map[string]string, i int, k string) {
	cKey := C.CString("")
	C.free(unsafe.Pointer(cKey))
	lenKey := len(m)
	if lenKey <= 0 {
		return
	}
	C.lua_createtable(s, 0, C.int(lenKey))
	for k2, v := range m {
		cKey = C.CString(v)
		C.lua_pushstring(s, cKey)
		C.free(unsafe.Pointer(cKey))
		cKey = C.CString(k2)
		C.lua_setfield(s, -2, cKey)
		C.free(unsafe.Pointer(cKey))
	}
	if i > 0 {
		C.lua_rawseti(s, -2, C.int(i))
	} else {
		cKey = C.CString(k)
		C.lua_setfield(s, -2, cKey);
		C.free(unsafe.Pointer(cKey))
	}
}

////////////////////////////////
func stateSetTableByMapList(s *C.lua_State, l []map[string]string, k string) {
	cKey := C.CString("")
	C.free(unsafe.Pointer(cKey))
	lenKey := len(l)
	if lenKey <= 0 {
		return
	}
	C.lua_createtable(s, C.int(lenKey), 0)
	for i := 0; i < lenKey; i ++ {
		stateSetTableByMap1(s, l[i], i+1, "")
	}
	cKey = C.CString(k)
	C.lua_setfield(s, -2, cKey);
	C.free(unsafe.Pointer(cKey))
}
	
	////////////////////////////////
func stateSetTableByMap2(s *C.lua_State, m map[string]map[string]string) {
	for k, _ := range m {
		stateSetTableByMap1(s, m[k], 0, k)
	}
}

////////////////////////////////
func stateApplySession(s *C.lua_State, session *DataSessionType) {
	// session
	C.lua_createtable(s, 0, 8)
	stateSetTableByMap1(s, session.Block, 0, "block")
	stateSetTableByMap1(s, session.Tx, 0, "tx")
	stateSetTableByMap1(s, session.Op, 0, "op")
	stateSetTableByMap1(s, session.OpScript, 0, "opScript")
	stateSetTableByMap1(s, session.ExData, 0, "exData")
	stateSetTableByMapList(s, session.TxInputs, "txInputs")
	stateSetTableByMapList(s, session.TxOutputs, "txOutputs")
	cKey := C.CString("session")
	C.lua_setfield(s, C.LUA_GLOBALSINDEX, cKey);
	C.free(unsafe.Pointer(cKey))
	// state
	C.lua_createtable(s, 0, C.int(len(session.State)))
	stateSetTableByMap2(s, session.State)
	cKey = C.CString("state")
	C.lua_setfield(s, C.LUA_GLOBALSINDEX, cKey);
	C.free(unsafe.Pointer(cKey))
}

////////////////////////////////
func stateGetDataToMap(s *C.lua_State, r *map[string]string) {
	if C.lua_type(s, -2) == C.LUA_TSTRING && C.lua_type(s, -1) == C.LUA_TSTRING {
		(*r)[C.GoString(C.lua_tolstring(s, -2, nil))] = C.GoString(C.lua_tolstring(s, -1, nil))
	}
}

////////////////////////////////
func stateGetDataToList(s *C.lua_State, r *[]string) {
	if C.lua_type(s, -2) == C.LUA_TNUMBER && C.lua_type(s, -1) == C.LUA_TSTRING {
		*r = append(*r, C.GoString(C.lua_tolstring(s, -1, nil)))
	}
}

////////////////////////////////
func stateGetTableData(s *C.lua_State, fn func()) {
	if C.lua_type(s, -1) != C.LUA_TTABLE {
		return
	}
	C.lua_pushnil(s)
	for C.lua_next(s, -2) != 0 {
		fn()
		C.lua_settop(s, C.lua_gettop(s)-1)
	}
}

////////////////////////////////
func stateGetTableMap1(s *C.lua_State, size int) (map[string]string) {
	if C.lua_type(s, -1) != C.LUA_TTABLE {
		return nil
	}
	result := make(map[string]string, size)
	stateGetTableData(s, func() {
		stateGetDataToMap(s, &result)
	})
	return result
}

////////////////////////////////
func stateGetTableMap2(s *C.lua_State, size1 int, size2 int) (map[string]map[string]string) {
	if C.lua_type(s, -1) != C.LUA_TTABLE {
		return nil
	}
	result := make(map[string]map[string]string, size1)
	key := ""
	stateGetTableData(s, func() {
		if C.lua_type(s, -2) == C.LUA_TSTRING && C.lua_type(s, -1) == C.LUA_TTABLE {
			key = C.GoString(C.lua_tolstring(s, -2, nil))
			data := make(map[string]string, size2)
			stateGetTableData(s, func() {
				stateGetDataToMap(s, &data)
				result[key] = data
			})
		}
	})
	return result
}

////////////////////////////////
func stateGetResult(s *C.lua_State) (*DataResultType, error) {
	if C.lua_type(s, -1) != C.LUA_TTABLE {
		return nil, fmt.Errorf("not a table @stateGetResult")
	}
	defer C.lua_settop(s, C.lua_gettop(s)-1)
	result := &DataResultType{}
	key := ""
	C.lua_pushnil(s)
	for C.lua_next(s, -2) != 0 {
		if C.lua_type(s, -2) != C.LUA_TSTRING {
			C.lua_settop(s, C.lua_gettop(s)-1)
			continue
		}
		key = C.GoString(C.lua_tolstring(s, -2, nil))
		if key == "op" {
			result.Op = stateGetTableMap1(s, 8)
		} else if key == "opScript" {
			result.OpScript = stateGetTableMap1(s, 16)
		} else if key == "opRules" {
			result.OpRules = stateGetTableMap1(s, 16)
		} else if key == "exData" {
			result.ExData = stateGetTableMap1(s, 8)
		} else if key == "keyRules" {
			result.KeyRules = stateGetTableMap1(s, 16)
		} else if key == "state" {
			result.State = stateGetTableMap2(s, 16, 8)
		}
		C.lua_settop(s, C.lua_gettop(s)-1)
	}
	return result, nil
}

// ...
