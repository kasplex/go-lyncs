
////////////////////////////////
package lyncs

//#cgo CFLAGS: -I/var/luajit2/src
//#cgo LDFLAGS: -L/var/luajit2/src -lluajit -ldl -lm -static
import "C"
import (

	// ...

)

////////////////////////////////
var lRuntime runtimeType

////////////////////////////////
func init() {
	lRuntime.cfg = &ConfigType{
		NumWorkers: 8,
		// ...
	}
	lRuntime.poolMap = make(map[string]*poolType)
	// ...
}

////////////////////////////////
func Config(cfg *ConfigType) {

	// validate cfg ...

	lRuntime.cfg = cfg
}

////////////////////////////////
func CodeVerify(code string) ([]byte, error) {
	bc := []byte{}

	// ...

	return bc, nil
}
