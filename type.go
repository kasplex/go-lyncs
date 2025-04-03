
////////////////////////////////
package lyncs

/*
#include "lua.h"
*/
import "C"

// ...

////////////////////////////////
type configType struct {
	nWorkers uint
	fBuiltin map[string]C.lua_CFunction

	// ...

}

////////////////////////////////
type poolType struct {
	idle map[int64]*C.lua_State
	inuse map[int64]*C.lua_State
	code string
	bc []byte
}

// ...

////////////////////////////////
type runtimeType struct {
	cfg *configType
	poolMap map[string]*poolType

	// ...

}
