
////////////////////////////////
package lyncs

//#cgo CFLAGS: -I/var/luajit2/src
//#cgo LDFLAGS: -L/var/luajit2/src -lluajit -ldl -lm -static
import "C"
import (
	"fmt"
	"sync"
)

////////////////////////////////
var lRuntime runtimeType

////////////////////////////////
func init() {
	lRuntime.cfg = &ConfigType{
		NumWorkers: 8,
		Callbacks: []string{"init", "run"},
		MaxInSlot: 128,
	}
	lRuntime.poolMap = make(map[string]*poolType)
	// ...
}

////////////////////////////////
func Config(cfg *ConfigType) {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = lRuntime.cfg.NumWorkers
	}
	if len(cfg.Callbacks) <= 0 {
		cfg.Callbacks = lRuntime.cfg.Callbacks
	}
	if cfg.MaxInSlot <= 0 {
		cfg.MaxInSlot = lRuntime.cfg.MaxInSlot
	}
	// ...
	lRuntime.cfg = cfg
}

////////////////////////////////
func CodeVerify(code string) ([]byte, error) {
	s, bc, err := stateFromCode(code)
	if err != nil {
		return nil, err
	}
	defer stateClose(s)
	for _, fn := range lRuntime.cfg.Callbacks {
		if !stateCheckFunc(s, fn) {
			return nil, fmt.Errorf("missing callback:" + fn + " @CodeVerify")
		}
	}
	return bc, nil
}

////////////////////////////////
func CallFuncParallel(callList []DataCallFuncType, stateMap map[string]map[string]string, fCallBefore func(*DataCallFuncType), fCallAfter func(*DataCallFuncType, *DataResultType, error)) ([]*DataResultType) {
	lenCall := len(callList)
	result := make([]*DataResultType, lenCall)
	iCall := 0
	slots := make([]dataCallSlotType, lRuntime.cfg.NumWorkers)
	for iCall < lenCall {
		for i, _ := range slots {
			slots[i].list = make([]int, 0, lRuntime.cfg.MaxInSlot)
			slots[i].keyRules = make(map[string]string, lRuntime.cfg.MaxInSlot / 4)
		}
		for i := iCall; i < lenCall; i ++ {
			iSlot := 0
			lenSlot := lRuntime.cfg.MaxInSlot
			conflict := false
			rwSwitch := false
			for j, _ := range slots {
				for key, rwCall := range callList[i].KeyRules {
					rwSlot, exists := slots[j].keyRules[key]
					if !exists {
						continue
					}
					if rwSlot == "w" {
						conflict = true
						break
					}
					if rwSlot == "r" && rwCall == "w" {
						rwSwitch = true
						break
					}
				}
				if conflict {
					lenSlot = len(slots[j].list)
					iSlot = j
					break
				}
				if rwSwitch {
					break
				}
				if len(slots[j].list) < lenSlot {
					lenSlot = len(slots[j].list)
					iSlot = j
				}
			}
			if rwSwitch || lenSlot >= lRuntime.cfg.MaxInSlot {
				iCall = i
				break
			}
			slots[iSlot].list = append(slots[iSlot].list, i)
			for key, rwCall := range callList[i].KeyRules {
				if rwCall == "w" && slots[iSlot].keyRules[key] != "w" || rwCall == "r" && slots[iSlot].keyRules[key] == "" {
					slots[iSlot].keyRules[key] = rwCall
				}
			}
			iCall = i + 1
		}
		var mutex sync.RWMutex
		wg := &sync.WaitGroup{}
		for i, _ := range slots {
			wg.Add(1)
			go func() {
				for _, j := range slots[i].list {
					mutex.RLock()
					for key, _ := range callList[j].KeyRules {
						callList[j].Session.State[key] = stateMap[key]
					}
					mutex.RUnlock()
					if fCallBefore != nil {
						fCallBefore(&callList[j])
					}
					r, err := PoolCallFunc(callList[j].Name, callList[j].Fn, callList[j].Session)
					if fCallAfter != nil {
						fCallAfter(&callList[j], r, err)
					}
					if r != nil && len(r.State) > 0 {
						for key, rw := range callList[j].KeyRules {
							if rw != "w" {
								continue
							}
							s, exists := r.State[key]
							if !exists {
								continue
							}
							if len(s) == 0 {
								s = nil
							}
							mutex.Lock()
							stateMap[key] = s
							mutex.Unlock()
						}
					}
					result[j] = r
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
	return result
}

// ...
