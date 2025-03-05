
////////////////////////////////
package lyncs

/*
#include "lua.h"
*/
import "C"
import (
    "fmt"
    "time"
    
    // ...
    
)

////////////////////////////////
func poolInit(key string) (error) {
    if key == "" {
        return fmt.Errorf("key empty @poolInit")
    }
    _, exists := lRuntime.poolMap[key]
    if exists {
        err := PoolDestroy(key)
        if err != nil {
            return err
        }
    }
    lRuntime.poolMap[key] = &poolType{
        idle: make(map[int64]*C.lua_State, lRuntime.cfg.nWorkers),
        inuse: make(map[int64]*C.lua_State, lRuntime.cfg.nWorkers),
    }
    return nil
}

////////////////////////////////
func PoolFromCode(key string, code string) (error) {
    err := poolInit(key)
    if err != nil {
        return err
    }
    s, err := stateFromCode(code)
    if err != nil {
        PoolDestroy(key)
        return err
    }
    lRuntime.poolMap[key].idle[time.Now().UnixNano()] = s
    lRuntime.poolMap[key].code = code
    return nil
}

////////////////////////////////
func PoolFromBuffer(key string, buffer []byte) (error) {
    err := poolInit(key)
    if err != nil {
        return err
    }
    s, err := stateFromBuffer(buffer)
    if err != nil {
        PoolDestroy(key)
        return err
    }
    lRuntime.poolMap[key].idle[time.Now().UnixNano()] = s
    lRuntime.poolMap[key].buffer = buffer
    return nil
}

////////////////////////////////
func PoolDestroy(key string) (error) {
    pool, exists := lRuntime.poolMap[key]
    if !exists {
        return nil
    }
    if len(pool.inuse) > 0 {
        return fmt.Errorf("pool exists/inuse @PoolDestroy")
    }
    for _, s := range pool.idle {
        stateClose(s)
    }
    delete(lRuntime.poolMap, key)
    return nil
}

////////////////////////////////
func PoolState(key string) (/*, */error) {

    // ...

    return /*, */nil
}

////////////////////////////////
func PoolList() (/*, */error) {

    // ...

    return /*, */nil
}
