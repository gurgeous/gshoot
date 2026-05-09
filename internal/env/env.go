//nolint:revive
package env

import "os"

//
// env vars used by gshoot. note that charm adds support for NO_COLOR and
// friends
//

var (
	// GOOGLE_APPLICATION_CREDENTIALS points to ADC credentials JSON.
	GOOGLE_APPLICATION_CREDENTIALS = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	// GSHOOT_CONFIG_DIR overrides the config directory.
	GSHOOT_CONFIG_DIR = os.Getenv("GSHOOT_CONFIG_DIR")
	// GSHOOT_CREDENTIALS_FILE points to Google credentials JSON.
	GSHOOT_CREDENTIALS_FILE = os.Getenv("GSHOOT_CREDENTIALS_FILE")
	// GSHOOT_THEME forces light or dark terminal styling.
	GSHOOT_THEME = os.Getenv("GSHOOT_THEME")
	// GSHOOT_TOKEN provides a raw OAuth access token.
	GSHOOT_TOKEN = os.Getenv("GSHOOT_TOKEN")
)
