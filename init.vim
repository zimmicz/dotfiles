" Specify a directory for plugins
" - For Neovim: ~/.local/share/nvim/plugged
" - Avoid using standard Vim directory names like 'plugin'
call plug#begin('~/.vim/plugged')

" Shorthand notation; fetches https://github.com/junegunn/vim-easy-align
Plug 'airblade/vim-gitgutter'
Plug 'https://github.com/itchyny/lightline.vim'
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' }
Plug 'junegunn/fzf.vim'
Plug 'scrooloose/nerdtree'
Plug 'tpope/vim-surround'
Plug 'https://github.com/vim-scripts/sql.vim--Stinson'
Plug 'terryma/vim-multiple-cursors'
Plug 'danilo-augusto/vim-afterglow'
Plug 'mhartington/oceanic-next'
Plug 'lifepillar/vim-solarized8'
Plug 'zanglg/nova.vim'

" Initialize plugin system
call plug#end()

let mapleader = "\<Space>"
syntax on
set expandtab
set autoindent
set smarttab
set tabstop=4
set shiftwidth=4
set softtabstop=4
set number
set laststatus=2
set cursorline
set title
set background=dark
set updatetime=100
set splitright

map ; :Files<CR>
map <C-o> :NERDTreeToggle<CR>

let g:lightline = {
  \     'active': {
  \         'left': [['mode', 'paste' ], ['readonly', 'filename', 'modified']],
  \         'right': [['lineinfo'], ['percent'], ['fileformat', 'fileencoding']]
  \     }
  \ }

" " Copy to clipboard
vnoremap  <leader>y  "+y
nnoremap  <leader>Y  "+yg_
nnoremap  <leader>y  "+y
nnoremap  <leader>yy  "+yy

" " Paste from clipboard
nnoremap <leader>p "+p
nnoremap <leader>P "+P
vnoremap <leader>p "+p
vnoremap <leader>P "+P

" " Move between splits
nnoremap <C-J> <C-W><C-J>
nnoremap <C-K> <C-W><C-K>
nnoremap <C-L> <C-W><C-L>
nnoremap <C-H> <C-W><C-H>

if (has("termguicolors"))
 set termguicolors
endif

" Theme
syntax enable
colorscheme nova

