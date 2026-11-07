# Made by Lex, which starts zsh with this folder as ZDOTDIR to style the
# prompt with Oh My Posh after your own configuration: these files load
# yours, from your ZDOTDIR or your home, and change none of them. Lex
# writes them anew as it starts a shell.
_lex_zdotdir=$ZDOTDIR
if [[ -n ${LEX_USER_ZDOTDIR+x} ]]; then ZDOTDIR=$LEX_USER_ZDOTDIR; else unset ZDOTDIR; fi
[[ -r ${ZDOTDIR:-$HOME}/.zshenv ]] && source ${ZDOTDIR:-$HOME}/.zshenv
# Your .zshenv may set ZDOTDIR: your other files are there.
if [[ -n ${ZDOTDIR+x} ]]; then LEX_USER_ZDOTDIR=$ZDOTDIR; else unset LEX_USER_ZDOTDIR; fi
ZDOTDIR=$_lex_zdotdir
unset _lex_zdotdir
