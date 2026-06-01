default:
  just --list

build:
  go build -o bin/gshoot
  just banner "✓ build ✓"

build-release:
  go build -ldflags "-w -s" -o bin/gshoot-release

clean:
  go clean -testcache
  rm -f bin/gshoot

format:
  go mod tidy
  golangci-lint fmt

install: build
  cp bin/gshoot ~/.local/bin

lint:
  golangci-lint run
  just banner "✓ lint ✓"

#
# run/test
#

check: lint build test test-bats
  just banner "✓ check ✓"

ci: check

run *ARGS: build
  gshoot {{ARGS}}

run-watch *ARGS:
  GSHOOT_THEME=1 watchexec -q --clear=reset just run {{ARGS}}

test *ARGS:
  go test ./... {{ARGS}}
  just banner "✓ test ✓"

test-bats *ARGS: build
  bats {{ARGS}} --print-output-on-failure testdata/smoke.bats
  just banner "✓ test-bats ✓"

test-live *ARGS: build
  bats {{ARGS}} --print-output-on-failure testdata/live.bats
  just banner "✓ test-live ✓"

test-watch *ARGS:
  GSHOOT_THEME=1 watchexec -q --clear=reset just test {{ARGS}}

#
# release
#

release *ARGS: check
  bin/release {{ARGS}}

release-preview: check
  goreleaser release --clean --snapshot --verbose
  just banner "tmp/dist"
  ls -lh tmp/dist
  just banner "cat tmp/dist/homebrew/gshoot.rb..."
  bat --paging=never tmp/dist/homebrew/gshoot.rb
  just banner "tarball..."
  tar -tvzf "$(find tmp/dist -maxdepth 1 -name '*_darwin_arm64.tar.gz' | head -n 1)"

#
# gmv
#

demo:
  go run . demo

demo-256:
  TERM=xterm-256color COLORTERM= TMUX= go run . demo

demo-true:
  COLORTERM=truecolor go run . demo

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
