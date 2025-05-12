
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
		idle: make(map[int64]*C.lua_State, lRuntime.cfg.NumWorkers),
		inuse: make(map[int64]*C.lua_State, lRuntime.cfg.NumWorkers),
	}
	return nil
}

////////////////////////////////
func PoolFromCode(key string, code string) (error) {
	err := poolInit(key)
	if err != nil {
		return err
	}
	s, bc, err := stateFromCode(code)
	if err != nil {
		PoolDestroy(key)
		return err
	}
	lRuntime.poolMap[key].idle[time.Now().UnixNano()] = s
	lRuntime.poolMap[key].code = code
	lRuntime.poolMap[key].bc = bc
	lRuntime.poolMap[key].top = C.lua_gettop(s)
	return nil
}

////////////////////////////////
func PoolFromBC(key string, bc []byte) (error) {
	err := poolInit(key)
	if err != nil {
		return err
	}
	s, err := stateFromBC(bc)
	if err != nil {
		PoolDestroy(key)
		return err
	}
	lRuntime.poolMap[key].idle[time.Now().UnixNano()] = s
	lRuntime.poolMap[key].bc = bc
	lRuntime.poolMap[key].top = C.lua_gettop(s)
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
func poolLockState(pool *poolType) (*C.lua_State, int64, error) {
	for i, s := range pool.idle {
		_, exists := pool.inuse[i]
		if !exists {
			pool.inuse[i] = s
			return s, i, nil
		}
	}
	if len(pool.idle) < lRuntime.cfg.NumWorkers {
		if pool.bc == nil {
			return nil, 0, fmt.Errorf("nil bytecode @poolLockState")
		}
		s, err := stateFromBC(pool.bc)
		if err != nil {
			return nil, 0, err
		}
		i := time.Now().UnixNano()
		pool.idle[i] = s
		pool.inuse[i] = s
		return s, i, nil
	}
	return nil, 0, fmt.Errorf("no available @poolLockState")
}

////////////////////////////////
func poolUnlockState(pool *poolType, index int64) {
	delete(pool.inuse, index)
}

////////////////////////////////
func PoolCallFunc(key string, f string, session *DataSessionType) (*DataResultType, error) {
	if key == "" {
		return nil, fmt.Errorf("key empty @PoolCallFunc")
	}
	pool, exists := lRuntime.poolMap[key]
	if !exists {
		return nil, fmt.Errorf("pool empty @PoolCallFunc")
	}
	s, index, err := poolLockState(pool)
	if err != nil {
		return nil, err
	}
	defer poolUnlockState(pool, index)
	stateClean(s, pool.top)
	stateApplySession(s, session)
	err = stateCallFunc(s, f, 1)
	if err != nil {
		return nil, err
	}
	result, err := stateGetResult(s)
	if err != nil {
		return nil, err
	}
	return result, nil
}

////////////////////////////////
func PoolList() (/*, */error) {

	// ...

	return /*, */nil
}
