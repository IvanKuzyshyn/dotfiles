" ==============================================================================
" Basic Settings
" ==============================================================================

" Enable syntax highlighting
syntax on

" Enable filetype detection and plugins
filetype plugin indent on

" ==============================================================================
" Display
" ==============================================================================

" Show line numbers
set number
set relativenumber

" Show ruler and command
set ruler
set showcmd

" Highlight current line
set cursorline

" ==============================================================================
" Search
" ==============================================================================

" Incremental search
set incsearch

" Highlight search results
set hlsearch

" Ignore case in search
set ignorecase

" Smart case search
set smartcase

" ==============================================================================
" Indentation
" ==============================================================================

" Auto indent
set autoindent
set smartindent

" Tab settings
set tabstop=4
set shiftwidth=4
set expandtab
set smarttab

" ==============================================================================
" Behavior
" ==============================================================================

" Enable mouse support
set mouse=a

" Show matching brackets
set showmatch

" Don't create backup files
set nobackup
set nowritebackup
set noswapfile

" Better command-line completion
set wildmenu
set wildmode=longest:full,full

" ==============================================================================
" Key Mappings
" ==============================================================================

" Set leader key to space
let mapleader = " "

" Clear search highlighting
nnoremap <leader>/ :nohlsearch<CR>

" Quick save
nnoremap <leader>w :w<CR>

" Quick quit
nnoremap <leader>q :q<CR>
