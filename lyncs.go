
////////////////////////////////
package lyncs

//#cgo CFLAGS: -I${SRCDIR}/luajit2/include
//#cgo LDFLAGS: -L${SRCDIR}/luajit2 -lluajit -ldl -lm -lgmp -lhashsum -lsha1 -lsha2 -lkeccak -lblake -static
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
func CallFuncParallel(callList []DataCallFuncType, stateMap map[string]map[string]string, mutex *sync.RWMutex, fCallBefore func(*DataCallFuncType), fCallAfter func(*DataCallFuncType, int, *DataResultType, error) (*DataResultType)) ([]*DataResultType) {
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
			var conflict bool
			var rwSwitch bool
            countConflict := 0
			for j, _ := range slots {
                conflict = false
                rwSwitch = false
				for key, rwCall := range callList[i].KeyRules {
					rwSlot, exists := slots[j].keyRules[key]
					if !exists {
						continue
					}
					if rwSlot == "w" {
						conflict = true
					}
					if rwSlot == "r" && rwCall == "w" {
						rwSwitch = true
                        break
					}
				}
				if rwSwitch {
					break
				}
				if conflict {
                    countConflict ++
                    if countConflict == 1 {
                        lenSlot = len(slots[j].list)
                        iSlot = j
                    } else {
                        rwSwitch = true
                        break
                    }
                    continue
				}
				if countConflict == 0 && len(slots[j].list) < lenSlot {
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
		if mutex == nil {
			mutex = &sync.RWMutex{}
		}
		wg := &sync.WaitGroup{}
		for i, _ := range slots {
			wg.Add(1)
			go func(i int) {
				for _, j := range slots[i].list {
					if callList[j].Session.State == nil {
						callList[j].Session.State = make(map[string]map[string]string, len(callList[j].KeyRules))
					}
					mutex.RLock()
					for key, _ := range callList[j].KeyRules {
						callList[j].Session.State[key] = stateMap[key]
					}
					mutex.RUnlock()
					if fCallBefore != nil {
						fCallBefore(&callList[j])
					}
					r, err := PoolCallFunc(callList[j].Name, callList[j].Fn, callList[j].Session)
					if r != nil && len(r.State) > 0 {
						for k := range r.State {
							//if s["_key"] == "" || callList[j].KeyRules[s["_key"]] != "w" {
							if callList[j].KeyRules[k] != "w" {
								r.State[k] = nil
								continue
							}
						}
					}
					if r == nil && err == nil {
						err = fmt.Errorf("nil result")
					}
					if fCallAfter != nil {
						r = fCallAfter(&callList[j], j, r, err)
					}
					result[j] = r
					if r == nil {
						continue
					}
					for k, s := range r.State {
						if s == nil {
							continue
						}
						//if len(s) == 1 {
						if len(s) == 0 {
							mutex.Lock()
							//stateMap[s["_key"]] = nil
							stateMap[k] = nil
							mutex.Unlock()
							continue
						}
						mutex.Lock()
						//stateMap[s["_key"]] = s
						stateMap[k] = s
						//delete(stateMap[s["_key"]], "_key")
						mutex.Unlock()
					}
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
	}
	return result
}

// ...
