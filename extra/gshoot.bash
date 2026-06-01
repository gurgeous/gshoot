declare -F _init_completion >/dev/null || return 2>/dev/null

_gshoot() {
  local cur prev words cword
  local command command_index auth_subcommand auth_subcommand_index
  local commands=(auth down up list peek wipe)
  local auth_commands=(login logout status)
  _init_completion || return

  _gshoot_find_command

  case "${prev}" in
    --client-secret)
      _gshoot_files json
      return
      ;;
    --output|-o)
      _gshoot_files csv
      return
      ;;
    --sheet)
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
      auth)
        _gshoot_find_auth_subcommand

        case "${auth_subcommand}" in
          login) _gshoot_words "--client-secret --help" ;;
          *) _gshoot_words "--help" ;;
        esac
        ;;
      down) _gshoot_words "-o --output --help" ;;
      up) _gshoot_words "--sheet --refill --replace --filter --layout --numeric --open --help" ;;
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
    auth)
      _gshoot_find_auth_subcommand

      if [[ -z "${auth_subcommand}" ]]; then
        _gshoot_words "${auth_commands[*]}"
      else
        COMPREPLY=()
      fi
      ;;
    up)
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

_gshoot_find_auth_subcommand() {
  local i

  auth_subcommand=
  auth_subcommand_index=0

  for ((i = command_index + 1; i < cword; i++)); do
    _gshoot_contains_word "${words[i]}" "${auth_commands[@]}" || continue
    auth_subcommand="${words[i]}"
    auth_subcommand_index="${i}"
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
