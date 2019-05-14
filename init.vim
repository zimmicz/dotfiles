" Specify a directory for plugins
" - For Neovim: ~/.local/share/nvim/plugged
" - Avoid using standard Vim directory names like 'plugin'
call plug#begin('~/.vim/plugged')

Plug 'airblade/vim-gitgutter'
Plug 'https://github.com/itchyny/lightline.vim'
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' }
Plug 'junegunn/fzf.vim'
Plug 'tpope/vim-surround'
Plug 'vim-airline/vim-airline'
Plug 'vim-airline/vim-airline-themes'
Plug 'jdkanani/vim-material-theme'
Plug 'kshenoy/vim-signature'
Plug 'jiangmiao/auto-pairs'
Plug 'rhysd/clever-f.vim'
Plug 'gabrielelana/vim-markdown'
Plug 'easymotion/vim-easymotion'
Plug 'pangloss/vim-javascript'
Plug 'othree/javascript-libraries-syntax.vim'
Plug 'burnettk/vim-angular'
Plug 'vim-scripts/DeleteTrailingWhitespace'
Plug 'vim-scripts/ShowTrailingWhitespace'
Plug 'w0rp/ale'
Plug 'tpope/vim-fugitive'
Plug 'tpope/vim-obsession'
Plug 'tpope/vim-unimpaired'
Plug 'tiagofumo/vim-nerdtree-syntax-highlight'
Plug 'kaicataldo/material.vim'
Plug 'scrooloose/nerdtree'
Plug 'editorconfig/editorconfig-vim'
Plug 'bdauria/angular-cli.vim'
Plug 'heavenshell/vim-jsdoc'
Plug 'alexandre/toggle_width.vim'
Plug 'Yggdroot/indentLine'
Plug 'posva/vim-vue'
Plug 'rakr/vim-one'
Plug 'leafgarland/typescript-vim'
Plug 'nvie/vim-flake8'
Plug 'majutsushi/tagbar'
" https://medium.com/@rahul11061995/autocomplete-in-vim-for-js-developer-698c6275e341
Plug 'Valloric/YouCompleteMe', { 'do': './install.py --tern-completer' }
Plug 'prettier/vim-prettier', {
    \ 'do': 'npm install',
    \ 'for': ['javascript', 'typescript', 'css', 'less', 'scss'] }

" Initialize plugin system
call plug#end()

let mapleader = "\<Space>"
let python_highlight_all=1
syntax on
set expandtab
set splitbelow
set splitright
set autoindent
set smarttab
set tabstop=4
set shiftwidth=4
set softtabstop=4
set number
set laststatus=2
set cursorline!
set lazyredraw
set title
set background=dark
set updatetime=750
set splitright
set clipboard+=unnamedplus
set nobackup
set noswapfile
set smartcase
set foldmethod=indent
set foldlevel=99

map <leader>; :GFiles<CR>
map <leader>e :NERDTreeToggle<CR>
imap jj <Esc>
nmap j gj
nmap k gk

" Use ; for commands.
nnoremap ; :
" Use Q to execute default register.
nnoremap Q @q

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

noremap <Up> <nop>
noremap <Down> <nop>
noremap <Left> <nop>
noremap <Right> <nop>

if (has("termguicolors"))
 set termguicolors
endif

if &term =~ '256color'
  " disable Background Color Erase (BCE) so that color schemes
  " render properly when inside 256-color tmux and GNU screen.
  set t_ut=
endif

" Theme
syntax enable
" colorscheme material
colorscheme one
set background=dark
set timeoutlen=1000 ttimeoutlen=0

let g:airline_powerline_fonts = 1
" let g:airline_theme = 'material'
let g:airline_theme='one'
cnoremap <C-a>  <Home>
cnoremap <C-b>  <Left>
cnoremap <C-f>  <Right>
cnoremap <C-d>  <Delete>
cnoremap <M-b>  <S-Left>
cnoremap <M-f>  <S-Right>
cnoremap <M-d>  <S-right><Delete>
cnoremap <Esc>b <S-Left>
cnoremap <Esc>f <S-Right>
cnoremap <Esc>d <S-right><Delete>
cnoremap <C-g>  <C-c>

autocmd BufReadPre *.js let b:javascript_lib_use_angularjs = 1

" Go to tab by number
noremap <leader>1 1gt
noremap <leader>2 2gt
noremap <leader>3 3gt
noremap <leader>4 4gt
noremap <leader>5 5gt
noremap <leader>6 6gt
noremap <leader>7 7gt
noremap <leader>8 8gt
noremap <leader>9 9gt
noremap <leader>0 :tablast<cr>
noremap <leader><leader>p :tabprevious<cr>

let g:DeleteTrailingWhitespace = 1
let g:DeleteTrailingWhitespace_Action = 'delete'

let g:airline#extensions#ale#enabled = 1

" vv to generate new vertical split
nnoremap <silent> vv <C-w>v

" sudo write
ca w!! w !sudo tee >/dev/null "%"

" close NERDTree if it is the only window left open
autocmd bufenter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif
autocmd FileType javascript setlocal omnifunc=javascriptcomplete#CompleteJS

" Start autocompletion after 4 chars
let g:ycm_min_num_of_chars_for_completion = 4
let g:ycm_min_num_identifier_candidate_chars = 4
let g:ycm_enable_diagnostic_highlighting = 0
" Don't show YCM's preview window [ I find it really annoying ]
set completeopt-=preview
let g:ycm_add_preview_to_completeopt = 0

let g:ale_linters = {
\   'javascript': ['eslint'],
\}

let g:ale_cache_executable_check_failures = 1

autocmd BufRead,BufNewFile *.vue setlocal filetype=vue.html.javascript.css

" Try to fix syntax highlighting
autocmd FileType vue syntax sync fromstart
noremap <F12> <Esc>:syntax sync fromstart<CR>
inoremap <F12> <C-o>:syntax sync fromstart<CR>

" Enable folding with the <Tab>
nnoremap <Tab> za
" Clear highlights with Ctrl+Shift+L
nnoremap <silent> <C-Space> :nohlsearch<CR>
" Buffer switch
nnoremap <silent> sql :set syntax=sql<CR>

" TagBar
nmap <F8> :TagbarToggle<CR>

" Buffers
nnoremap <leader>b :ls<CR>:b<Space>
" History
nnoremap <silent> <leader>/ :History<CR>

