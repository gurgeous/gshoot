#!/usr/bin/env bats

# setup hook
setup() {
  ROOT="$BATS_TEST_DIRNAME/.."
  BIN="$ROOT/bin/gshoot"
  export HOME="$BATS_TEST_TMPDIR/home"
  export GSHOOT_SMOKE=true
  mkdir -p "$HOME"
}

#
# helpers
#

login() {
  "$BIN" auth login --client-secret "$ROOT/testdata/oauth-client.json" >/dev/null
}

#
# tests
#

@test "welcome" {
  run "$BIN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"welcome"* ]]
  [[ "$output" == *"gshoot auth status"* ]]
}

@test "login (no secrets)" {
  run "$BIN" auth login
  [ "$status" -eq 0 ]
  [[ "$output" == *"gshoot auth status"* ]]
  [[ "$output" == *"client secrets file"* ]]
  [ ! -f "$HOME/.config/gshoot/oauth-token.json" ]
}

@test "login --client-secret" {
  run "$BIN" auth login --client-secret "$ROOT/testdata/oauth-client.json"
  [ "$status" -eq 0 ]
  [[ "$output" == *"success! oauth token copied"* ]]
  [ -f "$HOME/.config/gshoot/oauth-client.json" ]
  [ -f "$HOME/.config/gshoot/oauth-token.json" ]
}

@test "auth status" {
  login
  run "$BIN" auth status
  [ "$status" -eq 0 ]
  [[ "$output" == *"Client secrets file:"* ]]
  [[ "$output" == *"Token file:"* ]]
}

@test "no auth - show status" {
  run "$BIN" list
  [[ "$output" == *"you must authenticate first"* ]]
  run "$BIN" down smoke
  [[ "$output" == *"you must authenticate first"* ]]
  run "$BIN" up smoke bogus.csv
  [[ "$output" == *"you must authenticate first"* ]]
}

@test "no token - show status" {
  login
  rm "$HOME/.config/gshoot/oauth-token.json"

  run "$BIN" list
  [[ "$output" == *"complete \`gshoot auth login\` first"* ]]
  run "$BIN" down smoke
  [[ "$output" == *"complete \`gshoot auth login\` first"* ]]
  run "$BIN" up smoke bogus.csv
  [[ "$output" == *"complete \`gshoot auth login\` first"* ]]
}
