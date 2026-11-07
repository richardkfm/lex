# Made by Lex: loads your .zshrc, then Oh My Posh in Lex's theme (see
# .zshenv). ZDOTDIR is yours again from here on, for your .zlogin and the
# shells this one starts.
if [[ -n ${LEX_USER_ZDOTDIR+x} ]]; then ZDOTDIR=$LEX_USER_ZDOTDIR; else unset ZDOTDIR; fi
unset LEX_USER_ZDOTDIR
[[ -r ${ZDOTDIR:-$HOME}/.zshrc ]] && source ${ZDOTDIR:-$HOME}/.zshrc
if (( $+commands[oh-my-posh] )); then
  eval "$(oh-my-posh init zsh --config "$LEX_OMP_CONFIG")"
fi
