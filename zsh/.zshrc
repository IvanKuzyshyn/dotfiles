# ==============================================================================
# Oh My Zsh Configuration
# ==============================================================================

export ZSH="$HOME/.oh-my-zsh"

# Plugins (zsh-syntax-highlighting must be last)
plugins=(
    zsh-autosuggestions
    zsh-syntax-highlighting
)

# Load oh-my-zsh
[ -f "$ZSH/oh-my-zsh.sh" ] && source "$ZSH/oh-my-zsh.sh"

# ==============================================================================
# Environment Variables
# ==============================================================================

export EDITOR=vim
export VISUAL=vim
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8

# ==============================================================================
# Path Configuration
# ==============================================================================

# Go
export GOPATH="$HOME/go"
export PATH="$GOPATH/bin:$PATH"

# Local bin
export PATH="$HOME/.local/bin:$PATH"

# Homebrew
if [[ -d "/opt/homebrew" ]]; then
    export PATH="/opt/homebrew/bin:/opt/homebrew/sbin:$PATH"
fi

# ==============================================================================
# nvm Setup
# ==============================================================================

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"

# ==============================================================================
# Rust Setup
# ==============================================================================

[ -s "$HOME/.cargo/env" ] && \. "$HOME/.cargo/env"

# ==============================================================================
# Aliases - Kubernetes
# ==============================================================================

alias k='kubectl'
alias kgp='kubectl get pods'
alias kgs='kubectl get services'
alias kgd='kubectl get deployments'
alias kgn='kubectl get nodes'
alias kdp='kubectl describe pod'
alias kds='kubectl describe service'
alias kdd='kubectl describe deployment'
alias kl='kubectl logs'
alias klf='kubectl logs -f'
alias kex='kubectl exec -it'
alias kctx='kubectl config current-context'
alias kns='kubectl config set-context --current --namespace'

# ==============================================================================
# Aliases - Git
# ==============================================================================

alias g='git'
alias gs='git status'
alias ga='git add'
alias gc='git commit'
alias gco='git checkout'
alias gb='git branch'
alias gp='git push'
alias gpl='git pull'
alias gd='git diff'
alias gl='git log --oneline --graph --decorate'
alias gst='git stash'

# ==============================================================================
# Aliases - Directory & Files
# ==============================================================================

alias ll='ls -lah'
alias la='ls -A'
alias l='ls -CF'
alias ..='cd ..'
alias ...='cd ../..'
alias ....='cd ../../..'

# ==============================================================================
# Tool Completions
# ==============================================================================

# AWS CLI completion
if command -v aws_completer &> /dev/null; then
    autoload bashcompinit && bashcompinit
    complete -C aws_completer aws
fi

# GitHub CLI completion
if command -v gh &> /dev/null; then
    eval "$(gh completion -s zsh)"
fi

# kubectl completion
if command -v kubectl &> /dev/null; then
    source <(kubectl completion zsh)
    complete -F __start_kubectl k
fi

# ==============================================================================
# Prompt Configuration
# ==============================================================================

# Enable vcs_info for git branch in prompt
autoload -Uz vcs_info
precmd() { vcs_info }

# Configure vcs_info
zstyle ':vcs_info:git:*' formats ' (%b)'
zstyle ':vcs_info:*' enable git

setopt PROMPT_SUBST

# Simple prompt with git branch
PROMPT='%F{cyan}%~%f%F{yellow}${vcs_info_msg_0_}%f %# '

# ==============================================================================
# History Configuration
# ==============================================================================

HISTFILE=~/.zsh_history
HISTSIZE=10000
SAVEHIST=10000
setopt SHARE_HISTORY
setopt HIST_IGNORE_DUPS
setopt HIST_IGNORE_SPACE

# ==============================================================================
# Completion System
# ==============================================================================

autoload -Uz compinit && compinit

# Case-insensitive completion
zstyle ':completion:*' matcher-list 'm:{a-zA-Z}={A-Za-z}'

# ==============================================================================
# Key Bindings
# ==============================================================================

# Use emacs-style key bindings
bindkey -e