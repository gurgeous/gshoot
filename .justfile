default:
  just --list

build:
  go build -o bin/gshoot

run *ARGS: build
  gshoot {{ARGS}}

#
# hygiene
#

check:
  just lint && just banner "✓ lint ✓"
  just test && just banner "✓ test ✓"
  just build && just banner "✓ build ✓"
  just banner "✓ check ✓"

ci: check

format:
  go mod tidy
  golangci-lint fmt

lint:
  golangci-lint run

test:
  go test ./...

run-watch *ARGS:
  GSHOOT_THEME=1 watchexec -q --clear=reset just run {{ARGS}}

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
