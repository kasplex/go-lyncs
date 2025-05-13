
////////////////////////////////
package lyncs

//#cgo CFLAGS: -I/var/luajit2/src
//#cgo LDFLAGS: -L/var/luajit2/src -lluajit -ldl -lm -static
import "C"
import (
	"fmt"
)

////////////////////////////////
var lRuntime runtimeType

////////////////////////////////
func init() {
	lRuntime.cfg = &ConfigType{
		NumWorkers: 8,
		Callbacks: []string{"init", "run"},
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
	for _, f := range lRuntime.cfg.Callbacks {
		if !stateCheckFunc(s, f) {
			return nil, fmt.Errorf("missing callback:" + f + " @CodeVerify")
		}
	}
	return bc, nil
}

// ...
