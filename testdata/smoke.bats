#!/usr/bin/env bats

setup() {
  ROOT="$BATS_TEST_DIRNAME/.."
  BIN="$ROOT/bin/gshoot"
}

@test "bare command shows help" {
  run "$BIN"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Commands:"* ]]
  [[ "$output" == *"up"* ]]
  [[ "$output" == *"join"* ]]
  [[ "$output" != *"auth login"* ]]
  [[ "$output" != *"welcome"* ]]
}

@test "help describes spreadsheet references" {
  run "$BIN" down --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Spreadsheet name, ID, or URL."* ]]
}

@test "command aliases" {
  run "$BIN" d --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"down (d)"* ]]

  run "$BIN" u --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"up (u)"* ]]

  run "$BIN" ls --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"list (ls)"* ]]
}

@test "missing gog has an actionable error" {
  run env PATH=/nonexistent "$BIN" list
  [ "$status" -eq 1 ]
  [[ "$output" == *"gog is required"* ]]
  [[ "$output" == *"openclaw/tap/gogcli"* ]]
}

@test "zsh completion lists spreadsheet commands" {
  run zsh -fc '
    function compdef() {}
    function _arguments() { state=command; return 1 }
    function _describe() { print -rl -- "${commands[@]}" }
    source '"$ROOT"'/extra/_gshoot
    _gshoot
  '
  [ "$status" -eq 0 ]
  [[ "$output" == *"up:Upload a CSV"* ]]
  [[ "$output" == *"join:Join a CSV"* ]]
  [[ "$output" != *"auth:"* ]]
}

@test "zsh completion offers join flags" {
  run zsh -fc '
    function compdef() {}
    function _arguments() { print -rl -- "$@" }
    source '"$ROOT"'/extra/_gshoot
    _gshoot_join
  '
  [ "$status" -eq 0 ]
  [[ "$output" == *"--key[Column used to match rows]"* ]]
  [[ "$output" == *"--columns[Comma-separated CSV columns to join]"* ]]
  [[ "$output" == *":csv:_files"* ]]
}
