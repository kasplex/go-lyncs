
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
)

////////////////////////////////
var stateRemoveMap = map[string][]string{
	"string": {"dump"},
	"math": {"randomseed","random"},
	"_G": {"jit","collectgarbage","rawget","rawset","loadfile","load","loadstring","dofile","gcinfo","coroutine","debug","getfenv","newproxy","pcall","xpcall"},
}

var stateReadonlyList = `
	table = _set(table)
	string = _set(string)
	math = _set(math)
	bit = _set(bit)
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
	_, err := stateSandbox(s)
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
	for t, f := range lRuntime.cfg.Builtin {
		stateReadonlyList += "\t" + t + " = _set("+ t +")\r\n"
		codeSandbox += f + "\r\n"
	}
	codeCallbacks := "local f = {"
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
		local p = {}
		local mt = {
			__index = t,
			__newindex = function(_, k, v)
				if t ~= _G then
					error("variable read-only", 2)
				end
				if f[k] and t[k]==nil and type(v)=="function" then
					t[k] = v
					return
				end
				error("variable read-only", 2)
			end
		}
		setmetatable(p, mt)
		return p
	end
` + codeDebug + `
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
func stateCall(s *C.lua_State, nResult C.int) (error) {
	if C.LUA_OK != C.lua_pcall(s, 0, nResult, 0) {
		return stateError(s, "stateCall")
	}
	return nil
}

////////////////////////////////
func stateCallFunc(s *C.lua_State, f string, nResult C.int) (error) {
	cFunc := C.CString(f)
	defer C.free(unsafe.Pointer(cFunc))
	C.lua_getfield(s, C.LUA_GLOBALSINDEX, cFunc)
	return stateCall(s, nResult)
}

////////////////////////////////
func stateCheckFunc(s *C.lua_State, f string) (bool) {
	cFunc := C.CString(f)
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
func stateClean(s *C.lua_State, top C.int) {
	C.lua_settop(s, top)
	stateSetTableFieldNil(s, "_G", []string{"session", "state"})
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

////////////////////////////////
func stateApplySession(s *C.lua_State, session *DataSessionType) {
	////////////////////////////////
	_setTableByMap := func(m map[string]string, i int, k string) {
		cKey := C.CString("")
		C.free(unsafe.Pointer(cKey))
		lenKey := len(m)
		if lenKey <= 0 {
			return
		}
		C.lua_createtable(s, 0, C.int(lenKey))
		for k, v := range m {
			cKey = C.CString(v)
			C.lua_pushstring(s, cKey)
			C.free(unsafe.Pointer(cKey))
			cKey = C.CString(k)
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
	_setTableByMapList := func(l []map[string]string, k string) {
		cKey := C.CString("")
		C.free(unsafe.Pointer(cKey))
		lenKey := len(l)
		if lenKey <= 0 {
			return
		}
		C.lua_createtable(s, C.int(lenKey), 0)
		for i := 0; i < lenKey; i ++ {
			_setTableByMap(l[i], i+1, "")
		}
		cKey = C.CString(k)
		C.lua_setfield(s, -2, cKey);
		C.free(unsafe.Pointer(cKey))
	}
	////////////////////////////////
	_setTableMapState := func(m map[string]map[string]map[string]string) {
		cKey := C.CString("")
		C.free(unsafe.Pointer(cKey))
		lenKey := len(m)
		if lenKey <= 0 {
			return
		}
		for k, _ := range m {
			lenKey = len(m[k])
			if lenKey <= 0 {
				continue
			}
			C.lua_createtable(s, 0, C.int(lenKey))
			for k2, _ := range m[k] {
				_setTableByMap(m[k][k2], 0, k2)
			}
			cKey = C.CString(k)
			C.lua_setfield(s, -2, cKey);
			C.free(unsafe.Pointer(cKey))
		}
	}
	////////////////////////////////
	C.lua_createtable(s, 0, 8)
	_setTableByMap(session.Block, 0, "block")
	_setTableByMap(session.Tx, 0, "tx")
	_setTableByMap(session.Op, 0, "op")
	_setTableByMap(session.OpScript, 0, "opScript")
	_setTableByMap(session.ExData, 0, "exData")
	_setTableByMapList(session.TxInputs, "txInputs")
	_setTableByMapList(session.TxOutputs, "txOutputs")
	cKey := C.CString("session")
	C.lua_setfield(s, C.LUA_GLOBALSINDEX, cKey);
	C.free(unsafe.Pointer(cKey))
	////////////////////////////////
	C.lua_createtable(s, 0, 8)
	_setTableMapState(session.State)
	cKey = C.CString("state")
	C.lua_setfield(s, C.LUA_GLOBALSINDEX, cKey);
	C.free(unsafe.Pointer(cKey))
}

////////////////////////////////
func stateGetResult(s *C.lua_State) (*DataResultType, error) {
	////////////////////////////////
	_setResultToMap := func(r *map[string]string) {
		if C.lua_type(s, -2) == C.LUA_TSTRING && C.lua_type(s, -1) == C.LUA_TSTRING {
			(*r)[C.GoString(C.lua_tolstring(s, -2, nil))] = C.GoString(C.lua_tolstring(s, -1, nil))
		}
	}
	////////////////////////////////
	/*_setResultToList := func(r *[]string) {
		if C.lua_type(s, -2) == C.LUA_TNUMBER && C.lua_type(s, -1) == C.LUA_TSTRING {
			*r = append(*r, C.GoString(C.lua_tolstring(s, -1, nil)))
		}
	}*/
	////////////////////////////////
	_getTableData := func(f func()) {
		if C.lua_type(s, -1) != C.LUA_TTABLE {
			return
		}
		C.lua_pushnil(s)
		for C.lua_next(s, -2) != 0 {
			f()
			C.lua_settop(s, C.lua_gettop(s)-1)
		}
	}
	////////////////////////////////
	_getTableMap1 := func(size int) (map[string]string) {
		if C.lua_type(s, -1) != C.LUA_TTABLE {
			return nil
		}
		result := make(map[string]string, size)
		_getTableData(func() {
			_setResultToMap(&result)
		})
		return result
	}
	////////////////////////////////
	_getTableMap2 := func(size1 int, size2 int) (map[string]map[string]string) {
		if C.lua_type(s, -1) != C.LUA_TTABLE {
			return nil
		}
		result := make(map[string]map[string]string, size1)
		key := ""
		_getTableData(func() {
			if C.lua_type(s, -2) == C.LUA_TSTRING && C.lua_type(s, -1) == C.LUA_TTABLE {
				key = C.GoString(C.lua_tolstring(s, -2, nil))
				data := make(map[string]string, size2)
				_getTableData(func() {
					_setResultToMap(&data)
					result[key] = data
				})
			}
		})
		return result
	}
	////////////////////////////////
	_getTableMap3 := func(size1 int, size2 int, size3 int) (map[string]map[string]map[string]string) {
		if C.lua_type(s, -1) != C.LUA_TTABLE {
			return nil
		}
		result := make(map[string]map[string]map[string]string, size1)
		key1 := ""
		key2 := ""
		_getTableData(func() {
			if C.lua_type(s, -2) == C.LUA_TSTRING && C.lua_type(s, -1) == C.LUA_TTABLE {
				key1 = C.GoString(C.lua_tolstring(s, -2, nil))
				_getTableData(func() {
					if C.lua_type(s, -2) == C.LUA_TSTRING && C.lua_type(s, -1) == C.LUA_TTABLE {
						if result[key1] == nil {
							result[key1] = make(map[string]map[string]string, size2)
						}
						key2 = C.GoString(C.lua_tolstring(s, -2, nil))
						data := make(map[string]string, size3)
						_getTableData(func() {
							_setResultToMap(&data)
							result[key1][key2] = data
						})
					}
				})
			}
		})
		return result
	}
	////////////////////////////////
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
			result.Op = _getTableMap1(8)
		} else if key == "opScript" {
			result.OpScript = _getTableMap1(16)
		} else if key == "opRules" {
			result.OpRules = _getTableMap1(16)
		} else if key == "exData" {
			result.ExData = _getTableMap1(8)
		} else if key == "keyRules" {
			result.KeyRules = _getTableMap2(4, 4)
		} else if key == "state" {
			result.State = _getTableMap3(4, 4, 8)
		}
		C.lua_settop(s, C.lua_gettop(s)-1)
	}
	return result, nil
}

// ...
