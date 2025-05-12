
////////////////////////////////
package lyncs

/*
#include "lua.h"
*/
import "C"

////////////////////////////////
type ConfigType struct {
	NumWorkers int
	Callbacks []string
	Builtin map[string]string
	Debug bool
}

////////////////////////////////
type poolType struct {
	idle map[int64]*C.lua_State
	inuse map[int64]*C.lua_State
	code string
	bc []byte
	top C.int
}

// ...

////////////////////////////////
type runtimeType struct {
	cfg *ConfigType
	poolMap map[string]*poolType
	// ...
}

////////////////////////////////
type DataSessionType struct {
	Block map[string]string
	Tx map[string]string
	TxInputs []map[string]string
	TxOutputs []map[string]string
	Op map[string]string
	OpScript map[string]string
	State map[string]map[string]map[string]string
	// ...
	ExData map[string]string
}

////////////////////////////////
type DataResultType struct {
	Op map[string]string  // accept|error|feeLeast|isRecycle
	OpScript map[string]string
	OpRules map[string]string
	KeysRO map[string][]string
	KeysRW map[string][]string
	State map[string]map[string]map[string]string
	// ...
	ExData map[string]string
}

// ...
