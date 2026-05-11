default:
  just --list

build:
  go build -o bin/gshoot
  just banner "✓ build ✓"

#
# hygiene
#

check: lint test build
  just banner "✓ check ✓"

ci: check

clean:
  go clean -testcache
  rm -f bin/gshoot

format:
  go mod tidy
  golangci-lint fmt

lint:
  golangci-lint run
  just banner "✓ lint ✓"

run *ARGS: build
  gshoot {{ARGS}}

run-watch *ARGS:
  GSHOOT_THEME=1 watchexec -q --clear=reset just run {{ARGS}}

test *ARGS:
  go test ./... {{ARGS}}
  just banner "✓ test ✓"

test-watch *ARGS:
  GSHOOT_THEME=1 watchexec -q --clear=reset just test {{ARGS}}

#
# banner and friends
#

set quiet

[private]
banner msg bg="48;2;64;160;43":
  printf "\e[1;38;5;231;%sm[%s] %-72s\e[0m\n" "{{bg}}" $(date +"%H:%M:%S") "{{msg}}"

[private]
warning msg:
  just banner "{{msg}}" "48;2;251;100;11"

[private]
fatal msg:
  just banner "{{msg}}" "48;2;210;15;57"
  exit 1
