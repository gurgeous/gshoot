#!/usr/bin/env bats

setup() {
  ROOT="$BATS_TEST_DIRNAME/.."
  BIN="$ROOT/bin/gshoot"
  SHEET="gshoot-smoke"
}

banner() {
  # note fd3 for bats
  printf '\e[1;38;5;231;48;2;64;160;43m[%s] live: %-62s\e[0m\n' "$(date '+%H:%M:%S')" "$1" >&3
}

@test "live google workflow" {
  # verify auth before touching the scratch spreadsheet
  banner "preflight w/ list..."
  run "$BIN" list
  if [ "$status" -ne 0 ]; then
    echo "run gshoot auth login first" >&3
    return 1
  fi

  # reset the scratch spreadsheet to a single blank sheet
  banner "wipe $SHEET..."
  run "$BIN" wipe "$SHEET"
  [ "$status" -eq 0 ]
  [[ "$output" == *"wiped $SHEET"* ]]

  # confirm list and peek can see the reset spreadsheet
  banner "list..."
  run "$BIN" list
  [ "$status" -eq 0 ]
  [[ "$output" == *"$SHEET"* ]]

  banner "peek..."
  run "$BIN" peek "$SHEET"
  [ "$status" -eq 0 ]
  [[ "$output" == "Sheet1 "* ]]

  # replace upload should round-trip through download
  banner "up --replace..."
  printf 'name,score,city\nalice,1,denver\nbob,2,austin\n' >"$BATS_TEST_TMPDIR/basic.csv"
  run "$BIN" up --replace --sheet basic "$SHEET" "$BATS_TEST_TMPDIR/basic.csv"
  [ "$status" -eq 0 ]
  [[ "$output" == *"docs.google.com/spreadsheets/d/"* ]]

  banner "peek..."
  run "$BIN" peek "$SHEET"
  [ "$status" -eq 0 ]
  [[ "$output" == *"basic "* ]]

  banner "down..."
  run "$BIN" down -o "$BATS_TEST_TMPDIR/basic.out.csv" "$SHEET" basic
  [ "$status" -eq 0 ]
  grep -q "name,score,city" "$BATS_TEST_TMPDIR/basic.out.csv"
  grep -q "alice,1,denver" "$BATS_TEST_TMPDIR/basic.out.csv"
  grep -q "bob,2,austin" "$BATS_TEST_TMPDIR/basic.out.csv"

  # default upload should add a new sheet
  banner "up..."
  printf 'name,score,city\ncara,3,miami\ndrew,4,seattle\n' >"$BATS_TEST_TMPDIR/default.csv"
  run "$BIN" up --sheet default "$SHEET" "$BATS_TEST_TMPDIR/default.csv"
  [ "$status" -eq 0 ]

  banner "down..."
  run "$BIN" down -o "$BATS_TEST_TMPDIR/default.out.csv" "$SHEET" default
  [ "$status" -eq 0 ]
  grep -q "cara,3,miami" "$BATS_TEST_TMPDIR/default.out.csv"
  grep -q "drew,4,seattle" "$BATS_TEST_TMPDIR/default.out.csv"

  # numeric upload should still download stable values
  banner "up --numeric..."
  printf 'name,count\nalice,10\nbob,20\n' >"$BATS_TEST_TMPDIR/numeric.csv"
  run "$BIN" up --replace --numeric --sheet numeric "$SHEET" "$BATS_TEST_TMPDIR/numeric.csv"
  [ "$status" -eq 0 ]

  banner "down..."
  run "$BIN" down -o "$BATS_TEST_TMPDIR/numeric.out.csv" "$SHEET" numeric
  [ "$status" -eq 0 ]
  grep -q "alice,10" "$BATS_TEST_TMPDIR/numeric.out.csv"
  grep -q "bob,20" "$BATS_TEST_TMPDIR/numeric.out.csv"

  # refill should update rows and append new ones
  banner "up --refill..."
  printf 'id,name,count\na,Ada,1\nb,Bob,2\n' >"$BATS_TEST_TMPDIR/refill.csv"
  run "$BIN" up --replace --sheet refill "$SHEET" "$BATS_TEST_TMPDIR/refill.csv"
  [ "$status" -eq 0 ]

  banner "up --refill..."
  printf 'id,name,count\na,Ada,10\nb,Bob,20\nc,Cyd,30\n' >"$BATS_TEST_TMPDIR/refill.csv"
  run "$BIN" up --refill --sheet refill "$SHEET" "$BATS_TEST_TMPDIR/refill.csv"
  [ "$status" -eq 0 ]

  banner "down..."
  run "$BIN" down -o "$BATS_TEST_TMPDIR/refill.out.csv" "$SHEET" refill
  [ "$status" -eq 0 ]
  grep -q "a,Ada,10" "$BATS_TEST_TMPDIR/refill.out.csv"
  grep -q "b,Bob,20" "$BATS_TEST_TMPDIR/refill.out.csv"
  grep -q "c,Cyd,30" "$BATS_TEST_TMPDIR/refill.out.csv"
}
