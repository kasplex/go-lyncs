
////////////////////////////////
package lyncs

//#include "lua.h"
import "C"
import (
	"sync"
)

////////////////////////////////
type ConfigType struct {
	NumWorkers int
	Callbacks []string
	Builtin map[string]string
	MaxInSlot int
	Debug bool
}

////////////////////////////////
type poolType struct {
	sync.Mutex
	idle map[int64]*C.lua_State
	inuse map[int64]*C.lua_State
	code string
	bc []byte
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
	State map[string]map[string]string
	// ...
	ExData map[string]string
}

////////////////////////////////
type DataResultType struct {
	Op map[string]string  // accept|error|feeLeast|isRecycle
	OpScript map[string]string
	OpRules map[string]string
	KeyRules map[string]string
	State map[string]map[string]string
	// ...
	ExData map[string]string
}

////////////////////////////////
type DataCallFuncType struct {
	Name string
	Fn string
	Session *DataSessionType
	KeyRules map[string]string
}

////////////////////////////////
type dataCallSlotType struct {
	list []int
	keyRules map[string]string
}

// ...
