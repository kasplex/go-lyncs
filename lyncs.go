
////////////////////////////////
package lyncs

// ...

import (

	// ...

)

////////////////////////////////
var lRuntime runtimeType

////////////////////////////////
func init() {
	lRuntime.cfg = &configType{
		nWorkers: 8,
		// ...
	}
	lRuntime.poolMap = make(map[string]*poolType)
	// ...
}

////////////////////////////////
func Config(cfg *configType) {

	// validate cfg ...

	lRuntime.cfg = cfg
}

////////////////////////////////
func CodeVerify(code string) ([]byte, error) {
	bc := []byte{}

	// ...

	return bc, nil
}
