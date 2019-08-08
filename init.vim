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
Plug 'burnettk/vim-angular'
Plug 'vim-scripts/DeleteTrailingWhitespace'
Plug 'vim-scripts/ShowTrailingWhitespace'
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
Plug 'mileszs/ack.vim'
Plug 'tpope/vim-eunuch'
Plug 'tpope/vim-commentary'
Plug 'pangloss/vim-javascript'
Plug 'posva/vim-vue'
Plug 'mattn/emmet-vim'
Plug 'neoclide/coc.nvim', {'branch': 'release'}
let g:coc_global_extensions = ['coc-json', 'coc-tsserver', 'coc-emmet', 'coc-eslint']

" Initialize plugin system
call plug#end()

let mapleader = "\<Space>"
let python_highlight_all=1

" ----------------
" OPTIONS
" ----------------
syntax on
set autoindent
set autoread
set background=dark
set cindent
set cinkeys-=0#
set cino=
set clipboard+=unnamedplus
set commentstring=\ \ #%s   " When folds are created, add them to this
set copyindent
set cursorline!
set expandtab
set foldlevel=99
set foldmethod=indent
set hlsearch
set ignorecase
set incsearch
set laststatus=2
set lazyredraw
set nobackup
set noswapfile
set relativenumber
set shiftround
set shiftwidth=4
set smartcase
set smarttab
set softtabstop=4
set splitbelow
set splitright
set tabstop=4
set title
set updatetime=750
set formatoptions=tcqn1     " t - autowrap normal text
                            " c - autowrap comments
                            " q - gq formats comments
                            " n - autowrap lists
                            " 1 - break _before_ single-letter words
                            " 2 - use indenting from 2nd line of para
set hidden
set list                    " Show whitespace as special chars - see listchars
set listchars=tab:»\ ,extends:›,precedes:‹,nbsp:·,trail:· " Unicode characters for various things
set shortmess+=A

" Marks should go to the column, not just the line. Why isn't this the default?
nnoremap ' `

" Show git files
map <leader>; :GFiles --exclude-standard --others --cached<CR>

" Toggle nerd tree
map <leader>e :NERDTreeToggle<CR>

" Escape insert mode with double j
imap jj <Esc>

" Turn off linewise keys. Normally, the `j' and `k' keys move the cursor down one entire line. with
" line wrapping on, this can cause the cursor to actually skip a few lines on the screen because
" it's moving from line N to line N+1 in the file. I want this to act more visually -- I want `down'
" to mean the next line on the screen
nmap j gj
nmap k gk

" Open the last edited file.
nmap <C-e> :e#<CR>

" Use ; for commands.
nnoremap ; :
" Use Q to execute default register.
nnoremap Q @q

" Copy to clipboard
vnoremap  <leader>y  "+y
nnoremap  <leader>Y  "+yg_
nnoremap  <leader>y  "+y
nnoremap  <leader>yy  "+yy

" Paste from clipboard
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
let g:airline#extensions#branch#displayed_head_limit = 40
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

autocmd BufRead,BufNewFile *.vue setlocal filetype=vue.html.javascript.css

" Try to fix syntax highlighting
autocmd FileType vue syntax sync fromstart
noremap <F12> <Esc>:syntax sync fromstart<CR>
inoremap <F12> <C-o>:syntax sync fromstart<CR>

" Enable folding with the <Tab>
nnoremap <Tab> za
" Clear highlights with Ctrl+L
nnoremap <silent> <C-Space> :nohlsearch<CR>
" set sql highlighting
nnoremap <silent> sql :set syntax=sql<CR>
" TagBar
nmap <F8> :TagbarToggle<CR>

" Buffers
nnoremap <leader>b :ls<CR>:b<Space>
" History
nnoremap <silent> <leader>/ :History<CR>
" Reverse the functionality of Ack and Ack! (do not jump to the first result)
cnoreabbrev Ack Ack!
" Search the word under the cursor
nmap <leader>f   :Ack "\b<cword>\b" <CR>

let NERDTreeIgnore = ['\.js.map$', '__pycache__', '\.pyc$']

autocmd VimEnter * if globpath('.,..','node_modules/@angular') != '' | call angular_cli#init() | endif

" symap gf mapping
set inex=substitute(v:fname,'app','app/js/angular','')
set suffixesadd=.js,.ts,.scss

let g:tagbar_type_typescript = {
    \ 'ctagstype': 'typescript',
    \ 'kinds': [
      \ 'c:class',
      \ 'n:namespace',
      \ 'f:function',
      \ 'G:generator',
      \ 'v:variable',
      \ 'm:method',
      \ 'p:property',
      \ 'i:interface',
      \ 'g:enum',
      \ 't:type',
      \ 'a:alias',
    \ ],
    \'sro': '.',
      \ 'kind2scope' : {
      \ 'c' : 'class',
      \ 'n' : 'namespace',
      \ 'i' : 'interface',
      \ 'f' : 'function',
      \ 'G' : 'generator',
      \ 'm' : 'method',
      \ 'p' : 'property',
      \},
  \ }

" jump to newer location
:unmap <C-i>

" copy filename to clipboard
nnoremap <silent> cpfn :let @+=@%<CR>
