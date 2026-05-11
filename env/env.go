//nolint:revive
package env

import "os"

//
// env vars used by gshoot. note that charm adds support for NO_COLOR and
// friends
//

// GOOGLE_APPLICATION_CREDENTIALS points to ADC credentials JSON.
func GOOGLE_APPLICATION_CREDENTIALS() string { return os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") }

// GSHOOT_CONFIG_DIR overrides the config directory.
func GSHOOT_CONFIG_DIR() string { return os.Getenv("GSHOOT_CONFIG_DIR") }

// GSHOOT_CREDENTIALS_FILE points to Google credentials JSON.
func GSHOOT_CREDENTIALS_FILE() string { return os.Getenv("GSHOOT_CREDENTIALS_FILE") }

// GSHOOT_THEME forces light or dark terminal styling.
func GSHOOT_THEME() string { return os.Getenv("GSHOOT_THEME") }

// GSHOOT_TOKEN provides a raw OAuth access token.
func GSHOOT_TOKEN() string { return os.Getenv("GSHOOT_TOKEN") }
