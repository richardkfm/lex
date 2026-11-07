# Made by Lex: loads your .zprofile (see .zshenv).
_lex_zdotdir=$ZDOTDIR
if [[ -n ${LEX_USER_ZDOTDIR+x} ]]; then ZDOTDIR=$LEX_USER_ZDOTDIR; else unset ZDOTDIR; fi
[[ -r ${ZDOTDIR:-$HOME}/.zprofile ]] && source ${ZDOTDIR:-$HOME}/.zprofile
if [[ -n ${ZDOTDIR+x} ]]; then LEX_USER_ZDOTDIR=$ZDOTDIR; else unset LEX_USER_ZDOTDIR; fi
ZDOTDIR=$_lex_zdotdir
unset _lex_zdotdir
