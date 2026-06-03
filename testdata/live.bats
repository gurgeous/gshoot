#!/usr/bin/env bats

setup() {
  ROOT="$BATS_TEST_DIRNAME/.."
  BIN="$ROOT/bin/gshoot"
  SPREADSHEET="gshoot-smoke"
  SPREADSHEET_UPPER="GSHOOT-SMOKE"
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
  banner "wipe $SPREADSHEET..."
  run "$BIN" wipe -f "$SPREADSHEET"
  [ "$status" -eq 0 ]
  [[ "$output" == *"$SPREADSHEET"* ]]

  # confirm list and peek can see the reset spreadsheet
  banner "list..."
  run "$BIN" list
  [ "$status" -eq 0 ]
  [[ "$output" == *"$SPREADSHEET"* ]]

  banner "peek..."
  run "$BIN" peek "$SPREADSHEET"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Sheet1 "* ]]

  # spreadsheet file lookup should be case insensitive
  banner "peek uppercase..."
  run "$BIN" peek "$SPREADSHEET_UPPER"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Sheet1 "* ]]

  # replace upload should round-trip through download
  banner "up --replace..."
  printf 'name,score,city\nalice,1,denver\nbob,2,austin\n' >"$BATS_TEST_TMPDIR/basic.csv"
  run "$BIN" up --replace --sheet basic "$SPREADSHEET" "$BATS_TEST_TMPDIR/basic.csv"
  [ "$status" -eq 0 ]
  [[ "$output" == *"docs.google.com/spreadsheets/d/"* ]]

  banner "peek..."
  run "$BIN" peek "$SPREADSHEET"
  [ "$status" -eq 0 ]
  [[ "$output" == *"basic "* ]]

  banner "down..."
  run "$BIN" down --sheet basic -o "$BATS_TEST_TMPDIR/basic.out.csv" "$SPREADSHEET"
  [ "$status" -eq 0 ]
  grep -q "name,score,city" "$BATS_TEST_TMPDIR/basic.out.csv"
  grep -q "alice,1,denver" "$BATS_TEST_TMPDIR/basic.out.csv"
  grep -q "bob,2,austin" "$BATS_TEST_TMPDIR/basic.out.csv"

  # default upload should use the CSV basename as the sheet name
  banner "up..."
  printf 'name,score,city\ncara,3,miami\ndrew,4,seattle\n' >"$BATS_TEST_TMPDIR/default.csv"
  run "$BIN" up "$SPREADSHEET" "$BATS_TEST_TMPDIR/default.csv"
  [ "$status" -eq 0 ]

  banner "down..."
  run "$BIN" down --sheet default -o "$BATS_TEST_TMPDIR/default.out.csv" "$SPREADSHEET"
  [ "$status" -eq 0 ]
  grep -q "cara,3,miami" "$BATS_TEST_TMPDIR/default.out.csv"
  grep -q "drew,4,seattle" "$BATS_TEST_TMPDIR/default.out.csv"

  # numeric upload should still download stable values
  banner "up --numeric..."
  printf 'name,count\nalice,10\nbob,20\n' >"$BATS_TEST_TMPDIR/numeric.csv"
  run "$BIN" up --replace --numeric --sheet numeric "$SPREADSHEET" "$BATS_TEST_TMPDIR/numeric.csv"
  [ "$status" -eq 0 ]

  banner "down..."
  run "$BIN" down --sheet numeric -o "$BATS_TEST_TMPDIR/numeric.out.csv" "$SPREADSHEET"
  [ "$status" -eq 0 ]
  grep -q "alice,10" "$BATS_TEST_TMPDIR/numeric.out.csv"
  grep -q "bob,20" "$BATS_TEST_TMPDIR/numeric.out.csv"

  # refill should update rows and append new ones
  banner "up --refill..."
  printf 'id,name,count\na,Ada,1\nb,Bob,2\n' >"$BATS_TEST_TMPDIR/refill.csv"
  run "$BIN" up --replace --sheet refill "$SPREADSHEET" "$BATS_TEST_TMPDIR/refill.csv"
  [ "$status" -eq 0 ]

  banner "up --refill..."
  printf 'id,name,count\na,Ada,10\nb,Bob,20\nc,Cyd,30\n' >"$BATS_TEST_TMPDIR/refill.csv"
  run "$BIN" up --refill --sheet refill "$SPREADSHEET" "$BATS_TEST_TMPDIR/refill.csv"
  [ "$status" -eq 0 ]

  banner "down..."
  run "$BIN" down --sheet refill -o "$BATS_TEST_TMPDIR/refill.out.csv" "$SPREADSHEET"
  [ "$status" -eq 0 ]
  grep -q "a,Ada,10" "$BATS_TEST_TMPDIR/refill.out.csv"
  grep -q "b,Bob,20" "$BATS_TEST_TMPDIR/refill.out.csv"
  grep -q "c,Cyd,30" "$BATS_TEST_TMPDIR/refill.out.csv"
}
