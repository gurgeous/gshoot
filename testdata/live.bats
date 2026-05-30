#!/usr/bin/env bats

setup() {
  ROOT="$BATS_TEST_DIRNAME/.."
  BIN="$ROOT/bin/gshoot"
  SHEET="gshoot-smoke"
}

@test "live google workflow" {
  run "$BIN" list
  if [ "$status" -ne 0 ]; then
    echo "run gshoot auth login first" >&3
    return 1
  fi

  run "$BIN" wipe "$SHEET"
  [ "$status" -eq 0 ]
  [[ "$output" == *"wiped $SHEET"* ]]

  run "$BIN" peek "$SHEET"
  [ "$status" -eq 0 ]
  [[ "$output" == "Sheet1 "* ]]

  printf 'name,score,city\nalice,1,denver\nbob,2,austin\n' >"$BATS_TEST_TMPDIR/basic.csv"
  run "$BIN" up --replace --sheet basic "$SHEET" "$BATS_TEST_TMPDIR/basic.csv"
  [ "$status" -eq 0 ]
  [[ "$output" == *"docs.google.com/spreadsheets/d/"* ]]

  run "$BIN" peek "$SHEET"
  [ "$status" -eq 0 ]
  [[ "$output" == *"basic "* ]]

  run "$BIN" down -o "$BATS_TEST_TMPDIR/basic.out.csv" "$SHEET" basic
  [ "$status" -eq 0 ]
  grep -q "name,score,city" "$BATS_TEST_TMPDIR/basic.out.csv"
  grep -q "alice,1,denver" "$BATS_TEST_TMPDIR/basic.out.csv"
  grep -q "bob,2,austin" "$BATS_TEST_TMPDIR/basic.out.csv"

  printf 'name,count\nalice,10\nbob,20\n' >"$BATS_TEST_TMPDIR/numeric.csv"
  run "$BIN" up --replace --numeric --sheet numeric "$SHEET" "$BATS_TEST_TMPDIR/numeric.csv"
  [ "$status" -eq 0 ]

  run "$BIN" down -o "$BATS_TEST_TMPDIR/numeric.out.csv" "$SHEET" numeric
  [ "$status" -eq 0 ]
  grep -q "alice,10" "$BATS_TEST_TMPDIR/numeric.out.csv"
  grep -q "bob,20" "$BATS_TEST_TMPDIR/numeric.out.csv"
}
