
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
	"runtime"
	_ "embed"
)

//go:embed sandbox.lua
var luaSandbox string

////////////////////////////////
var bcSandbox []byte

////////////////////////////////
var stateRemoveMap = map[string][]string{
	"string": {"dump"},
	"math": {"randomseed","random"},
	"table": {"foreachi","foreach","getn","move","insert","remove"},
	"_G": {"jit","collectgarbage","rawget","rawset","rawequal","loadfile","load","loadstring","dofile","gcinfo","coroutine","debug","getfenv","setfenv","pcall","xpcall","newproxy","getmetatable"},
}

////////////////////////////////
func stateFromCode(code string) (*C.lua_State, []byte, error) {
    lenCode := len(code)
    if lenCode == 0 {
        return nil, nil, fmt.Errorf("empty code @stateFromCode")
    }
	s, err := stateSandbox()
	if err != nil {
		return nil, nil, err
	}
    r := C.luaL_loadbuffer(s, (*C.char)(unsafe.Pointer(unsafe.StringData(code))), C.size_t(lenCode), nil)
    runtime.KeepAlive(code)
    if r != C.LUA_OK {
		err = stateError(s, "stateFromCode")
		stateClose(s)
		return nil, nil, err
    }
	var bc []byte
	var buffer C.bcBuffer
	n := C.luaL_bcDump(s, &buffer)
	if n > 0 {
		bc = C.GoBytes(unsafe.Pointer(buffer.bc), C.int(n))
		C.free(unsafe.Pointer(buffer.bc))
	}
	stateEnvG(s)
	err = stateCall(s, 0)
	if err != nil {
		stateClose(s)
		return nil, nil, err
	}
    C.lua_gc(s, C.LUA_GCCOLLECT, 0)
	return s, bc, nil
}

////////////////////////////////
func stateFromBC(bc []byte) (*C.lua_State, error) {
    lenBC := len(bc)
    if lenBC == 0 {
        return nil, fmt.Errorf("nil bytecode @stateFromBC")
    }
	s, err := stateSandbox()
	if err != nil {
		return nil, err
	}
	r := C.luaL_loadbuffer(s, (*C.char)(unsafe.Pointer(&bc[0])), C.size_t(lenBC), nil)
    runtime.KeepAlive(bc)
    if r != C.LUA_OK {
		stateClose(s)
		return nil, fmt.Errorf("load failed @stateFromBC")
	}
	stateEnvG(s)
	err = stateCall(s, 0)
	if err != nil {
		stateClose(s)
		return nil, err
	}
    C.lua_gc(s, C.LUA_GCCOLLECT, 0)
	return s, nil
}

////////////////////////////////
func stateSandbox() (*C.lua_State, error) {
	s := C.luaL_newstate()
	if s == nil {
		return nil, fmt.Errorf("creation failed @stateSandbox")
	}
	var err error
	C.luaopen_base(s)
	C.luaopen_table(s)
	C.luaopen_string(s)
	C.luaopen_string_buffer(s)
	C.luaopen_math(s)
	C.luaopen_bit(s)
	C.luaopen_gmp(s, 256)
	C.luaopen_crypt(s)
	C.luaopen_jit(s)
	C.lua_settop(s, 0)
    C.lua_gc(s, C.LUA_GCSTOP, 0)
	for k, v := range stateRemoveMap {
		err = stateSetGlobalTableFieldNil(s, k, v)
		if err != nil {
			stateClose(s)
			return nil, err
		}
	}
	stateSetGlobalTableFieldString(s, "_G", []string{"_VERSION"}, []string{"LuaJIT 2.1 Lyncs"})
	if len(bcSandbox) > 0 {
		if C.LUA_OK != C.luaL_loadbuffer(s, (*C.char)(unsafe.Pointer(&bcSandbox[0])), C.size_t(len(bcSandbox)), nil) {
			stateClose(s)
			return nil, fmt.Errorf("load failed @stateSandbox")
		}
	} else {
		codeCallbacks := "\tlocal fn = {"
		if len(lRuntime.cfg.Callbacks) > 0 {
			for _, v := range lRuntime.cfg.Callbacks {
				codeCallbacks += `["`+v+`"]=true,`
			}
		}
		codeCallbacks += "}"
		codeDebug := "\tprint = function(...) end"
		if lRuntime.cfg.Debug {
			codeDebug = ""
		}
		codeSandbox := ""
		stateReadonlyList := ""
		for t, fn := range lRuntime.cfg.Builtin {
			stateReadonlyList += "\t" + t + " = _set("+ t +")\r\n"
			codeSandbox += fn + "\r\n"
		}
		codeSandbox += luaSandbox
		codeSandbox = strings.Replace(codeSandbox, "--[[-code-callbacks-]]", codeCallbacks, 1)
		codeSandbox = strings.Replace(codeSandbox, "--[[-code-readonly-list-]]", stateReadonlyList, 1)
		codeSandbox = strings.Replace(codeSandbox, "--[[-code-debug-]]", codeDebug, 1)
        r := C.luaL_loadbuffer(s, (*C.char)(unsafe.Pointer(unsafe.StringData(codeSandbox))), C.size_t(len(codeSandbox)), nil)
        runtime.KeepAlive(codeSandbox)
        if r != C.LUA_OK {
			err = stateError(s, "stateSandbox")
			stateClose(s)
			return nil, err
        }
		var buffer C.bcBuffer
		n := C.luaL_bcDump(s, &buffer)
		if n > 0 {
			bcSandbox = C.GoBytes(unsafe.Pointer(buffer.bc), C.int(n))
			C.free(unsafe.Pointer(buffer.bc))
		} else {
			stateClose(s)
			return nil, fmt.Errorf("bytecode failed @stateSandbox")
		}
	}
	err = stateCall(s, 0)
	if err != nil {
		stateClose(s)
		return nil, err
	}
	return s, nil
}

////////////////////////////////
func stateEnvG(s *C.lua_State) {
	cEnv := C.CString("_G")
	defer C.free(unsafe.Pointer(cEnv))
	C.lua_getfield(s, C.LUA_GLOBALSINDEX, cEnv)
	C.lua_setfenv(s, -2)
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
	if len(msgS[1]) > 40 {
		msgS[1] = msgS[1][:40] + ".."
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
    if int(C.lua_gc(s,C.LUA_GCCOUNT,0)) >= 8192 {
        C.lua_gc(s, C.LUA_GCCOLLECT, 0)
    }
}

////////////////////////////////
func stateSetGlobalTableField(s *C.lua_State, table string, field []string, fSet func(*C.lua_State, int)) (error) {
	ct := C.CString(table)
	defer C.free(unsafe.Pointer(ct))
	C.lua_getfield(s, C.LUA_GLOBALSINDEX, ct)
	defer C.lua_settop(s, C.lua_gettop(s)-1)
	if C.lua_type(s, -1) != C.LUA_TTABLE {
		return fmt.Errorf("not a table @stateSetGlobalTableField")
	}
	for i, fn := range field {
        C.lua_pushlstring(s, (*C.char)(unsafe.Pointer(unsafe.StringData(fn))), C.size_t(len(fn)))
        runtime.KeepAlive(fn)
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
        v := value[i]
        C.lua_pushlstring(s, (*C.char)(unsafe.Pointer(unsafe.StringData(v))), C.size_t(len(v)))
        runtime.KeepAlive(v)
	})
}

////////////////////////////////
func stateSetTableByMap1(s *C.lua_State, m map[string]string, i int, k string) {
    var cKey *C.char
	lenKey := len(m)
	if lenKey <= 0 {
		return
	}
	C.lua_createtable(s, 0, C.int(lenKey))
	for k2, v := range m {
        C.lua_pushlstring(s, (*C.char)(unsafe.Pointer(unsafe.StringData(v))), C.size_t(len(v)))
        runtime.KeepAlive(v)
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
	lenKey := len(l)
	if lenKey <= 0 {
		return
	}
	C.lua_createtable(s, C.int(lenKey), 0)
	for i := 0; i < lenKey; i ++ {
		stateSetTableByMap1(s, l[i], i+1, "")
	}
	cKey := C.CString(k)
	C.lua_setfield(s, -2, cKey);
	C.free(unsafe.Pointer(cKey))
}
	
	////////////////////////////////
func stateSetTableByMap2(s *C.lua_State, m map[string]map[string]string) {
	for k, v := range m {
		stateSetTableByMap1(s, v, 0, k)
	}
}

////////////////////////////////
func stateApplySession(s *C.lua_State, session *DataSessionType) {
	// session
	C.lua_createtable(s, 0, 8)
	stateSetTableByMap1(s, session.Block, 0, "block")
	stateSetTableByMap1(s, session.Tx, 0, "tx")
	stateSetTableByMap1(s, session.Op, 0, "op")
	stateSetTableByMap1(s, session.OpParams, 0, "opParams")
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
	stateGetTableData(s, func() {
		if C.lua_type(s, -2) == C.LUA_TSTRING && C.lua_type(s, -1) == C.LUA_TTABLE {
            key := C.GoString(C.lua_tolstring(s, -2, nil))
            data := make(map[string]string, size2)
			stateGetTableData(s, func() {
				stateGetDataToMap(s, &data)
			})
            result[key] = data
		}
	})
	return result
}

////////////////////////////////
func stateGetTableMapList(s *C.lua_State, size1 int, size2 int) ([]map[string]string) {
	if C.lua_type(s, -1) != C.LUA_TTABLE {
		return nil
	}
	result := make([]map[string]string, 0, size1)
	stateGetTableData(s, func() {
		if C.lua_type(s, -2) == C.LUA_TNUMBER && C.lua_type(s, -1) == C.LUA_TTABLE {
			data := make(map[string]string, size2)
			stateGetTableData(s, func() {
				stateGetDataToMap(s, &data)
			})
			result = append(result, data)
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
		} else if key == "opParams" {
			result.OpParams = stateGetTableMap1(s, 16)
		} else if key == "opRules" {
			result.OpRules = stateGetTableMap1(s, 16)
		} else if key == "exData" {
			result.ExData = stateGetTableMap1(s, 8)
		} else if key == "keyRules" {
			result.KeyRules = stateGetTableMap1(s, 16)
		} else if key == "state" {
			//result.State = stateGetTableMapList(s, 16, 8)
			result.State = stateGetTableMap2(s, 16, 8)
		}
		C.lua_settop(s, C.lua_gettop(s)-1)
	}
	return result, nil
}
