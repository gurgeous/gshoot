//nolint:revive
package env

import "os"

//
// environment variables used by gshoot. note that charm also looks at many env
// vars, like NO_COLOR, TERM, etc. Those are not listed here.
//

// GSHOOT_THEME forces light or dark terminal styling.
func GSHOOT_THEME() string { return os.Getenv("GSHOOT_THEME") }
