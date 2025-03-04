
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
    idle map[uint64]C.lua_State
    inuse map[uint64]C.lua_State
    code string
    buffer []byte
    
    // ...
    
}

// ...

////////////////////////////////
type runtimeType struct {
    cfg *configType
    vm map[string]*poolType
    
    // ...
    
}
