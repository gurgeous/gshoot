declare -F _init_completion >/dev/null || return 2>/dev/null

_gshoot() {
  local cur prev command subcommand
  _init_completion || return

  command="${words[1]}"
  subcommand="${words[2]}"

  case "${prev}" in
  --client-secret)
    _filedir json
    [[ ${#COMPREPLY[@]} -eq 0 ]] && _filedir
    return
    ;;
  --output | -o)
    _filedir csv
    [[ ${#COMPREPLY[@]} -eq 0 ]] && _filedir
    return
    ;;
  --sheet)
    COMPREPLY=()
    return
    ;;
  esac

  if [[ "${cur}" == -* ]]; then
    if [[ "${COMP_CWORD}" == 1 ]]; then
      COMPREPLY=($(compgen -W "--help --version" -- "${cur}"))
      return
    fi

    case "${command}" in
    auth)
      case "${subcommand}" in
      login) COMPREPLY=($(compgen -W "--client-secret --help" -- "${cur}")) ;;
      *) COMPREPLY=($(compgen -W "--help" -- "${cur}")) ;;
      esac
      ;;
    down) COMPREPLY=($(compgen -W "-o --output --help" -- "${cur}")) ;;
    up) COMPREPLY=($(compgen -W "--sheet --refill --replace --filter --layout --numeric --open --help" -- "${cur}")) ;;
    wipe) COMPREPLY=($(compgen -W "-f --force --help" -- "${cur}")) ;;
    *) COMPREPLY=($(compgen -W "--help" -- "${cur}")) ;;
    esac
    return
  fi

  case "${COMP_CWORD}" in
  1) COMPREPLY=($(compgen -W "auth down up list peek wipe" -- "${cur}")) ;;
  2)
    if [[ "${command}" == "auth" ]]; then
      COMPREPLY=($(compgen -W "login logout status" -- "${cur}"))
    else
      COMPREPLY=()
    fi
    ;;
  *)
    case "${command}" in
    up)
      _filedir '@(csv|tsv)'
      [[ ${#COMPREPLY[@]} -eq 0 ]] && _filedir
      ;;
    *) _filedir ;;
    esac
    ;;
  esac
}

complete -F _gshoot gshoot
