#!/usr/bin/env bats

#
# helpers
#

setup() {
  ROOT="$BATS_TEST_DIRNAME/.."
  BIN="$ROOT/bin/gshoot"
  F="gshoot-smoke"
  F_UPPER="GSHOOT-SMOKE"
  cd "$BATS_TEST_TMPDIR"
}

banner() {
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

write_file() {
  printf '%s' "$2" | sed '1{/^[[:space:]]*$/d}; s/^[[:space:]]*//; ${/^[[:space:]]*$/d;}' >"$1"
  printf '\n' >>"$1"
}

file_eq() {
  write_file want.txt "$1"
  normalize_file want.txt >want.csv
  normalize_file "$2" >got.csv
  diff -u want.csv got.csv >&3
}

file_eq_file() {
  normalize_file "$1" >want.csv
  normalize_file "$2" >got.csv
  if ! diff -u want.csv got.csv >&3; then
    echo "file mismatch: $1 != $2" >&3
    wc -c "$1" "$2" >&3
    return 1
  fi
}

normalize_file() {
  awk '
    {
      sub(/\r$/, "")
      sub(/[ \t]+$/, "")
      lines[NR] = $0
    }
    END {
      n = NR
      while (n > 0 && lines[n] == "") n--
      for (i = 1; i <= n; i++) print lines[i]
    }
  ' "$1"
}

#
# test
#

@test 'live google workflow (takes around 45s)' {
  # verify auth before touching the scratch spreadsheet
  banner "preflight w/ list..."
  run "$BIN" list
  if [ "$status" -ne 0 ]; then
    echo "run gshoot auth login first" >&3
    printf '%s\n' "$output" >&3
    return 1
  fi

  # reset scratch file
  run_ok wipe -f "$F" && [[ "$output" == *"$F"* ]]

  #
  # list/peek
  #

  run_ok list && [[ "$output" == *"$F"* ]]
  run_ok peek "$F" && [[ "$output" == *"Sheet1 "* ]]
  run_ok peek "$F_UPPER" && [[ "$output" == *"Sheet1 "* ]]

  #
  # up
  #

  write_file default.csv "
  name,score,city
  Cara,3,miami
  Drew,4,seattle
  "
  run_ok up "$F" default.csv && [[ "$output" == *"docs.google.com"* ]]
  run_ok down --sheet default -o default.out.csv "$F"
  file_eq_file default.csv default.out.csv

  #
  # up --sheet
  #

  write_file basic.csv "
  name,score,city
  Adam,1,denver
  Bob,2,austin
  "
  run_ok up --replace --sheet basic "$F" basic.csv && [[ "$output" == *"docs.google.com"* ]]
  run_ok peek "$F" && [[ "$output" == *"basic "* ]]
  run_ok down --sheet basic -o basic.out.csv "$F"
  file_eq_file basic.csv basic.out.csv

  #
  # up --refill grow
  #

  # seed
  write_file refill.csv "
  id,name,count
  a,Adam,1
  b,Bob,2
  "
  run_ok up --replace --sheet refill "$F" refill.csv
  # go
  write_file refill.csv "
  id,name,count
  a,Adam,10
  b,Bob,20
  c,Cara,30
  "
  run_ok up --refill --sheet refill "$F" refill.csv
  run_ok down --sheet refill -o refill.out.csv "$F"
  file_eq "
  id,name,count
  a,Adam,10
  b,Bob,20
  c,Cara,30
  " refill.out.csv

  #
  # up --refill shrink
  #

  # seed
  write_file refill-shrink.csv "
  id,name,count
  a,Adam,1
  b,Bob,2
  c,Cara,3
  "
  run_ok up --replace --sheet refill-shrink "$F" refill-shrink.csv
  # go
  write_file refill-shrink.csv "
  id,name,count
  a,Adam,10
  "
  run_ok up --refill --sheet refill-shrink "$F" refill-shrink.csv
  run_ok down --sheet refill-shrink -o refill-shrink.out.csv "$F"
  file_eq "
  id,name,count
  a,Adam,10
  " refill-shrink.out.csv

  #
  # up --refill don't touch remote rows
  #

  # seed
  write_file refill-keep.csv "
  id,KEEP,name
  a,KEEP Adam,Adam
  b,KEEP Bob,Bob
  c,KEEP Cara,Cara
  "
  run_ok up --replace --sheet refill-keep "$F" refill-keep.csv
  # go
  write_file refill-keep.csv "
  id,name
  a,Adam 10
  "
  run_ok up --refill --sheet refill-keep "$F" refill-keep.csv
  run_ok down --sheet refill-keep -o refill-keep.out.csv "$F"
  file_eq "
  id,KEEP,name
  a,KEEP Adam,Adam 10
  ,KEEP Bob,
  ,KEEP Cara,
  " refill-keep.out.csv

  #
  # up --refill shrink remote rows if blank
  #

  # seed
  write_file refill-blank-remote.csv "
  id,KEEP,name
  a,KEEP Adam,Adam
  b,,Bob
  c,,Cara
  "
  run_ok up --replace --sheet refill-blank-remote "$F" refill-blank-remote.csv
  # go
  write_file refill-blank-remote.csv "
  id,name
  a,Adam 10
  "
  run_ok up --refill --sheet refill-blank-remote "$F" refill-blank-remote.csv
  run_ok down --sheet refill-blank-remote -o refill-blank-remote.out.csv "$F"
  file_eq "
  id,KEEP,name
  a,KEEP Adam,Adam 10
  " refill-blank-remote.out.csv
}
