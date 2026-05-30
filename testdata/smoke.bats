#!/usr/bin/env bats

setup() {
  ROOT="$BATS_TEST_DIRNAME/.."
  BIN="$ROOT/bin/gshoot"
  export HOME="$BATS_TEST_TMPDIR/home"
  export GSHOOT_SMOKE=true
  mkdir -p "$HOME"
}

@test "auth flow" {
  run "$BIN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"welcome"* ]]
  [[ "$output" == *"gshoot auth status"* ]]
  [[ "$output" == *"client secrets file"* ]]

  run "$BIN" auth login --client-secret "$ROOT/testdata/oauth-client.json"
  [ "$status" -eq 0 ]
  [[ "$output" == *"copied to"* ]]
  [[ "$output" == *"success! oauth token copied"* ]]
  [ -f "$HOME/.config/gshoot/oauth-client.json" ]
  [ -f "$HOME/.config/gshoot/oauth-token.json" ]

  run "$BIN" auth status
  [ "$status" -eq 0 ]
  [[ "$output" == *"Client secrets file:"* ]]
  [[ "$output" == *"Token file:"* ]]
  [[ "$output" == *"present"* ]]
}
