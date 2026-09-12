declare -F _init_completion >/dev/null || return 2>/dev/null

_gshoot() {
  local cur prev words cword
  local command command_index
  local commands=(append down d hyperlink join up u list ls peek wipe)
  _init_completion || return

  _gshoot_find_command

  case "${prev}" in
    --output|-o)
      _gshoot_files csv
      return
      ;;
    --sheet|--key|--columns)
      COMPREPLY=()
      return
      ;;
  esac

  if [[ "${cur}" == -* ]]; then
    if [[ -z "${command}" || "${cword}" -le "${command_index}" ]]; then
      _gshoot_words "--help --version"
      return
    fi

    case "${command}" in
      append) _gshoot_words "--sheet --help" ;;
      down|d) _gshoot_words "-o --output --sheet --help" ;;
      hyperlink) _gshoot_words "--sheet --help" ;;
      join) _gshoot_words "--key --sheet --columns -f --force --help" ;;
      up|u) _gshoot_words "--sheet --refill --replace --filter --layout --numeric --open --help" ;;
      list|ls) _gshoot_words "--limit --help" ;;
      wipe) _gshoot_words "-f --force --help" ;;
      *) _gshoot_words "--help" ;;
    esac
    return
  fi

  if [[ -z "${command}" ]]; then
    _gshoot_words "${commands[*]}"
    return
  fi

  case "${command}" in
    append|up|u|join)
      if (( cword == command_index + 2 )); then
        _gshoot_files '@(csv|tsv)'
      else
        COMPREPLY=()
      fi
      ;;
    *)
      COMPREPLY=()
      ;;
  esac
}

_gshoot_contains_word() {
  local word="$1"
  local candidate
  shift

  for candidate in "$@"; do
    [[ "${candidate}" == "${word}" ]] && return 0
  done

  return 1
}

_gshoot_find_command() {
  local i

  command=
  command_index=0

  for ((i = 1; i < cword; i++)); do
    _gshoot_contains_word "${words[i]}" "${commands[@]}" || continue
    command="${words[i]}"
    command_index="${i}"
    return
  done
}

_gshoot_words() {
  COMPREPLY=($(compgen -W "$1" -- "${cur}"))
}

_gshoot_files() {
  _filedir "$1"
  [[ ${#COMPREPLY[@]} -eq 0 ]] && _filedir
}

complete -F _gshoot gshoot
