
////////////////////////////////
package lyncs

//#include "lua.h"
import "C"
import (
	"fmt"
	"time"
)

////////////////////////////////
func poolInit(name string) (error) {
	if name == "" {
		return fmt.Errorf("empty name @poolInit")
	}
    lRuntime.Lock()
	_, exists := lRuntime.poolMap[name]
	lRuntime.Unlock()
	if exists {
		err := PoolDestroy(name)
		if err != nil {
			return err
		}
	}
    lRuntime.Lock()
	lRuntime.poolMap[name] = &poolType{
		idle: make(map[int64]*C.lua_State, lRuntime.cfg.NumWorkers),
		inuse: make(map[int64]*C.lua_State, lRuntime.cfg.NumWorkers),
		cycle: make(map[int64]int, lRuntime.cfg.NumWorkers),
	}
    lRuntime.Unlock()
	return nil
}

////////////////////////////////
func PoolFromCode(name string, code string) (error) {
	err := poolInit(name)
	if err != nil {
		return err
	}
	s, bc, err := stateFromCode(code)
	if err != nil {
		PoolDestroy(name)
		return err
	}
	lRuntime.Lock()
    index := time.Now().UnixNano()
	lRuntime.poolMap[name].idle[index] = s
	lRuntime.poolMap[name].cycle[index] = 0
	lRuntime.poolMap[name].code = code
	lRuntime.poolMap[name].bc = bc
	lRuntime.Unlock()
	return nil
}

////////////////////////////////
func PoolFromBC(name string, bc []byte) (error) {
	err := poolInit(name)
	if err != nil {
		return err
	}
	s, err := stateFromBC(bc)
	if err != nil {
		PoolDestroy(name)
		return err
	}
	lRuntime.Lock()
    index := time.Now().UnixNano()
	lRuntime.poolMap[name].idle[index] = s
	lRuntime.poolMap[name].cycle[index] = 0
	lRuntime.poolMap[name].bc = bc
	lRuntime.Unlock()
	return nil
}

////////////////////////////////
func PoolDestroy(name string) (error) {
	lRuntime.Lock()
	defer lRuntime.Unlock()
	pool, exists := lRuntime.poolMap[name]
	if !exists {
		return nil
	}
	if len(pool.inuse) > 0 {
		return fmt.Errorf("pool exists/inuse @PoolDestroy")
	}
	for _, s := range pool.idle {
		stateClose(s)
	}
	delete(lRuntime.poolMap, name)
	return nil
}

////////////////////////////////
func poolLockState(pool *poolType) (*C.lua_State, int64, error) {
	pool.Lock()
	defer pool.Unlock()
	for i, s := range pool.idle {
		_, exists := pool.inuse[i]
		if !exists {
			pool.inuse[i] = s
			pool.cycle[i] ++
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
		pool.cycle[i] = 0
		return s, i, nil
	}
	return nil, 0, fmt.Errorf("no available @poolLockState")
}

////////////////////////////////
func poolUnlockState(pool *poolType, index int64) {
    var s *C.lua_State
	pool.Lock()
	delete(pool.inuse, index)
    if pool.cycle[index] >= 100000 {
        s = pool.idle[index]
        delete(pool.idle, index)
        delete(pool.cycle, index)
    }
    pool.Unlock()
    if s != nil {
        stateClose(s)
    }
}

////////////////////////////////
func PoolCallFunc(name string, fn string, session *DataSessionType) (*DataResultType, error) {
	if name == "" {
		return nil, fmt.Errorf("empty name @PoolCallFunc")
	}
	lRuntime.Lock()
	pool, exists := lRuntime.poolMap[name]
	lRuntime.Unlock()
	if !exists {
		return nil, fmt.Errorf("empty pool @PoolCallFunc")
	}
	if session == nil {
		return nil, fmt.Errorf("nil session @PoolCallFunc")
	}
	s, index, err := poolLockState(pool)
	if err != nil {
		return nil, err
	}
	defer poolUnlockState(pool, index)
	stateClean(s)
	stateApplySession(s, session)
	err = stateCallFunc(s, fn, 1)
	if err != nil {
		return nil, err
	}
	result, err := stateGetResult(s)
	if err != nil {
		return nil, err
	}
	return result, nil
}
