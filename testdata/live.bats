#!/usr/bin/env bats

#
# helpers
#

setup() {
  ROOT="$BATS_TEST_DIRNAME/.."
  BIN="$ROOT/bin/gshoot"
  F="gshoot-smoke"
  F_UPPER="GSHOOT-SMOKE"
}

banner() {
  # note fd3 for bats
  printf '\e[1;38;5;231;48;2;64;160;43m[%s] live: %-62s\e[0m\n' "$(date '+%H:%M:%S')" "$1" >&3
}

run_ok() {
  banner "$*..."
  run "$BIN" "$@"
  if [ "$status" -ne 0 ]; then
    printf 'command failed:' >&3
    printf ' %q' "$BIN" "$@" >&3
    printf '\nstatus: %s\n%s\n' "$status" "$output" >&3
    return 1
  fi
}

grep_missing() {
  if grep -q "$1" "$2"; then
    echo "unexpected match: $1" >&3
    return 1
  fi
}

#
# test
#

@test "live google workflow" {
  # verify auth before touching the scratch spreadsheet
  banner "preflight w/ list..."
  run "$BIN" list
  if [ "$status" -ne 0 ]; then
    echo "run gshoot auth login first" >&3
    return 1
  fi

  # reset scratch file
  run_ok wipe -f "$F"
  [[ "$output" == *"$F"* ]]

  #
  # list/peek
  #

  run_ok list && [[ "$output" == *"$F"* ]]
  run_ok peek "$F" && [[ "$output" == *"Sheet1 "* ]]
  run_ok peek "$F_UPPER" && [[ "$output" == *"Sheet1 "* ]]

  #
  # up w/o --sheet
  #

  printf 'name,score,city\ncara,3,miami\ndrew,4,seattle\n' >"$BATS_TEST_TMPDIR/default.csv"
  run_ok up "$F" "$BATS_TEST_TMPDIR/default.csv"
  # check
  run_ok down --sheet default -o "$BATS_TEST_TMPDIR/default.out.csv" "$F"
  grep -q "cara,3,miami" "$BATS_TEST_TMPDIR/default.out.csv"
  grep -q "drew,4,seattle" "$BATS_TEST_TMPDIR/default.out.csv"

  #
  # up --replace w/ --sheet
  #

  printf 'name,score,city\nalice,1,denver\nbob,2,austin\n' >"$BATS_TEST_TMPDIR/basic.csv"
  run_ok up --replace --sheet basic "$F" "$BATS_TEST_TMPDIR/basic.csv"
  [[ "$output" == *"docs.google.com/spreadsheets/d/"* ]]
  run_ok peek "$F"
  [[ "$output" == *"basic "* ]]
  # check
  run_ok down --sheet basic -o "$BATS_TEST_TMPDIR/basic.out.csv" "$F"
  grep -q "name,score,city" "$BATS_TEST_TMPDIR/basic.out.csv"
  grep -q "alice,1,denver" "$BATS_TEST_TMPDIR/basic.out.csv"
  grep -q "bob,2,austin" "$BATS_TEST_TMPDIR/basic.out.csv"

  #
  # up --numeric
  #

  printf 'name,count\nalice,10\nbob,20\n' >"$BATS_TEST_TMPDIR/numeric.csv"
  run_ok up --replace --numeric --sheet numeric "$F" "$BATS_TEST_TMPDIR/numeric.csv"
  # check
  run_ok down --sheet numeric -o "$BATS_TEST_TMPDIR/numeric.out.csv" "$F"
  grep -q "alice,10" "$BATS_TEST_TMPDIR/numeric.out.csv"
  grep -q "bob,20" "$BATS_TEST_TMPDIR/numeric.out.csv"

  #
  # up --refill grow
  #

  # seed
  printf 'id,name,count\na,Ada,1\nb,Bob,2\n' >"$BATS_TEST_TMPDIR/refill.csv"
  run_ok up --replace --sheet refill "$F" "$BATS_TEST_TMPDIR/refill.csv"
  # go
  printf 'id,name,count\na,Ada,10\nb,Bob,20\nc,Cyd,30\n' >"$BATS_TEST_TMPDIR/refill.csv"
  run_ok up --refill --sheet refill "$F" "$BATS_TEST_TMPDIR/refill.csv"
  # check
  run_ok down --sheet refill -o "$BATS_TEST_TMPDIR/refill.out.csv" "$F"
  grep -q "a,Ada,10" "$BATS_TEST_TMPDIR/refill.out.csv"
  grep -q "b,Bob,20" "$BATS_TEST_TMPDIR/refill.out.csv"
  grep -q "c,Cyd,30" "$BATS_TEST_TMPDIR/refill.out.csv"

  #
  # up --refill shrink
  #

  # seed
  printf 'id,name,count\na,Ada,1\nb,Bob,2\nc,Cyd,3\n' >"$BATS_TEST_TMPDIR/refill-shrink.csv"
  run_ok up --replace --sheet refill-shrink "$F" "$BATS_TEST_TMPDIR/refill-shrink.csv"
  # go
  printf 'id,name,count\na,Ada,10\n' >"$BATS_TEST_TMPDIR/refill-shrink.csv"
  run_ok up --refill --sheet refill-shrink "$F" "$BATS_TEST_TMPDIR/refill-shrink.csv"
  # check stale rows gone
  run_ok down --sheet refill-shrink -o "$BATS_TEST_TMPDIR/refill-shrink.out.csv" "$F"
  grep -q "a,Ada,10" "$BATS_TEST_TMPDIR/refill-shrink.out.csv"
  grep_missing "b,Bob,2" "$BATS_TEST_TMPDIR/refill-shrink.out.csv"
  grep_missing "c,Cyd,3" "$BATS_TEST_TMPDIR/refill-shrink.out.csv"

  #
  # up --refill shrink
  #

  # seed
  printf 'id,note,name\na,keep Ada,Ada\nb,keep Bob,Bob\nc,keep Cyd,Cyd\n' >"$BATS_TEST_TMPDIR/refill-keep.csv"
  run_ok up --replace --sheet refill-keep "$F" "$BATS_TEST_TMPDIR/refill-keep.csv"
  # go
  printf 'id,name\na,Ada 10\n' >"$BATS_TEST_TMPDIR/refill-keep.csv"
  run_ok up --refill --sheet refill-keep "$F" "$BATS_TEST_TMPDIR/refill-keep.csv"
  # check remote rows kept
  run_ok down --sheet refill-keep -o "$BATS_TEST_TMPDIR/refill-keep.out.csv" "$F"
  grep -q "a,keep Ada,Ada 10" "$BATS_TEST_TMPDIR/refill-keep.out.csv"
  grep -q ",keep Bob," "$BATS_TEST_TMPDIR/refill-keep.out.csv"
  grep -q ",keep Cyd," "$BATS_TEST_TMPDIR/refill-keep.out.csv"
  grep_missing "b,keep Bob,Bob" "$BATS_TEST_TMPDIR/refill-keep.out.csv"
  grep_missing "c,keep Cyd,Cyd" "$BATS_TEST_TMPDIR/refill-keep.out.csv"

  #
  # up --refill shrink
  #

  # seed
  printf 'id,note,name\na,keep Ada,Ada\nb,,Bob\nc,,Cyd\n' >"$BATS_TEST_TMPDIR/refill-blank-remote.csv"
  run_ok up --replace --sheet refill-blank-remote "$F" "$BATS_TEST_TMPDIR/refill-blank-remote.csv"
  # go
  printf 'id,name\na,Ada 10\n' >"$BATS_TEST_TMPDIR/refill-blank-remote.csv"
  run_ok up --refill --sheet refill-blank-remote "$F" "$BATS_TEST_TMPDIR/refill-blank-remote.csv"
  # check blank remote-cols shrink
  run_ok down --sheet refill-blank-remote -o "$BATS_TEST_TMPDIR/refill-blank-remote.out.csv" "$F"
  grep -q "a,keep Ada,Ada 10" "$BATS_TEST_TMPDIR/refill-blank-remote.out.csv"
  grep_missing "b,,Bob" "$BATS_TEST_TMPDIR/refill-blank-remote.out.csv"
  grep_missing "c,,Cyd" "$BATS_TEST_TMPDIR/refill-blank-remote.out.csv"
}
