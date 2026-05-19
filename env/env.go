//nolint:revive
package env

import "os"

//
// environment variables used by gshoot. note that charm also looks at many env
// vars, like NO_COLOR, TERM, etc. Those are not listed here.
//

func GSHOOT_THEME() string              { return os.Getenv("GSHOOT_THEME") }
func GSHOOT_GMV_FPS() string            { return os.Getenv("GSHOOT_GMV_FPS") }
func GSHOOT_GMV_WIDTH() string          { return os.Getenv("GSHOOT_GMV_WIDTH") }
func GSHOOT_GMV_HEIGHT() string         { return os.Getenv("GSHOOT_GMV_HEIGHT") }
func GSHOOT_GMV_ALPHA() string          { return os.Getenv("GSHOOT_GMV_ALPHA") }
func GSHOOT_GMV_DIFF_THRESHOLD() string { return os.Getenv("GSHOOT_GMV_DIFF_THRESHOLD") }
