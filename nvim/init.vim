" coc-vetur needs https://www.npmjs.com/package/vue-language-server
" see https://github.com/neoclide/coc-vetur/issues/28 when coc-vetur not working
call plug#begin('~/.vim/plugged')
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' }
Plug 'junegunn/fzf.vim',
Plug 'editorconfig/editorconfig-vim'
Plug 'rhysd/clever-f.vim'
Plug 'mileszs/ack.vim'
Plug 'easymotion/vim-easymotion'
Plug 'airblade/vim-gitgutter'
Plug 'itchyny/lightline.vim'
Plug 'drewtempelmeyer/palenight.vim'
Plug 'vim-scripts/DeleteTrailingWhitespace'
Plug 'vim-scripts/ShowTrailingWhitespace'
Plug 'tpope/vim-fugitive'
Plug 'tpope/vim-unimpaired'
Plug 'scrooloose/nerdtree'
Plug 'Xuyuanp/nerdtree-git-plugin'
"Plug 'tiagofumo/vim-nerdtree-syntax-highlight'
Plug 'peitalin/vim-jsx-typescript'
Plug 'preservim/nerdcommenter'
Plug 'alexandre/toggle_width.vim'
Plug 'Yggdroot/indentLine'
Plug 'sheerun/vim-polyglot'
Plug 'mhinz/vim-startify'
Plug 'SirVer/ultisnips'
Plug 'mlaursen/vim-react-snippets'
Plug 'posva/vim-vue'
Plug 'adelarsq/vim-matchit'
Plug 'neoclide/coc.nvim', {'branch': 'release'}
Plug 'antoinemadec/coc-fzf', {'branch': 'release'}
Plug 'ryanoasis/vim-devicons'
Plug 'styled-components/vim-styled-components', { 'branch': 'main' }
Plug 'kshenoy/vim-signature'
let g:coc_global_extensions = [
    \ 'coc-emmet',
    \ 'coc-eslint',
    \ 'coc-json',
    \ 'coc-pairs',
    \ 'coc-snippets',
    \ 'coc-stylelintplus',
    \ 'coc-tsserver',
    \ 'coc-vetur'
    \ ]
call plug#end()

" sudo apt install python3-setuptools python3-pip
" pip3 install --user wheel neovim
let g:python3_host_prog = '/usr/bin/python3'

let mapleader = "\<Space>"

" Escape insert mode with double j
imap jj <Esc>

" Use ; for commands.
nnoremap ; :
" Use Q to execute default register.
nnoremap Q @q

" Copy to clipboard
vnoremap  <leader>y  "+y
nnoremap  <leader>Y  "+yg_
nnoremap  <leader>yy  "+yy

" Paste from clipboard
nnoremap <leader>p "+p
nnoremap <leader>P "+P

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
syntax on
colorscheme palenight
set background=dark
set timeoutlen=1000 ttimeoutlen=0
set autoindent
set autoread
set background=dark
set cindent
set cinkeys-=0#
set cino=
set clipboard+=unnamedplus " sudo apt install xclip
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

" Turn off linewise keys. Normally, the `j' and `k' keys move the cursor down one entire line. with
" line wrapping on, this can cause the cursor to actually skip a few lines on the screen because
" it's moving from line N to line N+1 in the file. I want this to act more visually -- I want `down'
" to mean the next line on the screen
nmap j gj
nmap k gk

" Toggle nerd tree
map <leader>e :NERDTreeToggle<CR>
nmap <leader>n :NERDTreeFind<CR>

" Marks should go to the column, not just the line. Why isn't this the default?
nnoremap ' `

" Show git files
map <leader>; :GFiles --exclude-standard --others --cached<CR>

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

" close NERDTree if it is the only window left open
autocmd bufenter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif
autocmd FocusGained,BufEnter * :silent! !

" Try to fix syntax highlighting
"autocmd FileType vue syntax sync fromstart
noremap <F12> <Esc>:syntax sync fromstart<CR>
inoremap <F12> <C-o>:syntax sync fromstart<CR>

" Enable folding with the <Tab>
" nnoremap <Tab> za
" Clear highlights with Ctrl+L
nnoremap <silent> <C-Space> :nohlsearch<CR>
" set sql highlighting
nnoremap <silent> sql :set syntax=sql<CR>
" TagBar
nmap <F8> :TagbarToggle<CR>


" Command history
nnoremap q: :History:<CR>
" Search history
nnoremap q/ :History/<CR>
" Buffers
nnoremap <leader>b :Buffers<CR>
" History
nnoremap <silent> <leader>/ :History<CR>
" Reverse the functionality of Ack and Ack! (do not jump to the first result)
cnoreabbrev Ack Ack!
" Search the word under the cursor
nmap <leader>f   :Ack "\b<cword>\b" <CR>
" Git
nnoremap <leader>g :G<CR>

let NERDTreeIgnore = ['\.js.map$', '__pycache__', '\.pyc$', 'node_modules', 'offline']
let g:NERDTreeGitStatusIndicatorMapCustom = {
    \ "Modified"  : "✹",
    \ "Staged"    : "✚",
    \ "Untracked" : "✭",
    \ "Renamed"   : "➜",
    \ "Unmerged"  : "═",
    \ "Deleted"   : "✖",
    \ "Dirty"     : "✗",
    \ "Clean"     : "✔︎",
    \ 'Ignored'   : '☒',
    \ "Unknown"   : "?"
    \ }

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

" copy filename to clipboard
nnoremap <silent> cpfn :let @+=@%<CR>
nnoremap <silent> cpcfn :let @+=expand('%:p')<CR>

" coc settings
set updatetime=300
set signcolumn=yes
set shortmess+=c
" Use tab for trigger completion with characters ahead and navigate.
" NOTE: Use command ':verbose imap <tab>' to make sure tab is not mapped by
" other plugin before putting this into your config.
inoremap <silent><expr> <TAB>
      \ pumvisible() ? "\<C-n>" :
      \ <SID>check_back_space() ? "\<TAB>" :
      \ coc#refresh()
inoremap <expr><S-TAB> pumvisible() ? "\<C-p>" : "\<C-h>"

function! s:check_back_space() abort
  let col = col('.') - 1
  return !col || getline('.')[col - 1]  =~# '\s'
endfunction

" Use <c-space> to trigger completion.
inoremap <silent><expr> <c-space> coc#refresh()

" Use <cr> to confirm completion, `<C-g>u` means break undo chain at current
" position. Coc only does snippet and additional edit on confirm.
if has('patch8.1.1068')
  " Use `complete_info` if your (Neo)Vim version supports it.
  inoremap <expr> <cr> complete_info()["selected"] != "-1" ? "\<C-y>" : "\<C-g>u\<CR>"
else
  imap <expr> <cr> pumvisible() ? "\<C-y>" : "\<C-g>u\<CR>"
endif

" Use `[g` and `]g` to navigate diagnostics
nmap <silent> [g <Plug>(coc-diagnostic-prev)
nmap <silent> ]g <Plug>(coc-diagnostic-next)

" GoTo code navigation.
nmap <silent> gd <Plug>(coc-definition)
nmap <silent> gy <Plug>(coc-type-definition)
nmap <silent> gi <Plug>(coc-implementation)
nmap <silent> gr <Plug>(coc-references)

" Use K to show documentation in preview window.
nnoremap <silent> K :call <SID>show_documentation()<CR>

function! s:show_documentation()
  if (index(['vim','help'], &filetype) >= 0)
    execute 'h '.expand('<cword>')
  else
    call CocAction('doHover')
  endif
endfunction

" Highlight the symbol and its references when holding the cursor.
autocmd CursorHold * silent call CocActionAsync('highlight')

" Symbol renaming.
nmap <leader>rn <Plug>(coc-rename)

augroup mygroup
  autocmd!
  " Setup formatexpr specified filetype(s).
  autocmd FileType typescript,json setl formatexpr=CocAction('formatSelected')
  " Update signature help on jump placeholder.
  autocmd User CocJumpPlaceholder call CocActionAsync('showSignatureHelp')
augroup end

" Applying codeAction to the selected region.
" Example: `<leader>aap` for current paragraph
xmap <leader>a  <Plug>(coc-codeaction-selected)
nmap <leader>a  <Plug>(coc-codeaction-selected)

" Remap keys for applying codeAction to the current line.
nmap <leader>ac  <Plug>(coc-codeaction)
" Apply AutoFix to problem on the current line.
nmap <leader>qf  <Plug>(coc-fix-current)

" Introduce function text object
" NOTE: Requires 'textDocument.documentSymbol' support from the language server.
xmap if <Plug>(coc-funcobj-i)
xmap af <Plug>(coc-funcobj-a)
omap if <Plug>(coc-funcobj-i)
omap af <Plug>(coc-funcobj-a)

" Use <TAB> for selections ranges.
" NOTE: Requires 'textDocument/selectionRange' support from the language server.
" coc-tsserver, coc-python are the examples of servers that support it.
nmap <silent> <TAB> <Plug>(coc-range-select)
xmap <silent> <TAB> <Plug>(coc-range-select)

" Add `:Format` command to format current buffer.
command! -nargs=0 Format :call CocAction('format')

" Add `:Fold` command to fold current buffer.
command! -nargs=? Fold :call     CocAction('fold', <f-args>)

" Add `:OR` command for organize imports of the current buffer.
command! -nargs=0 OR   :call     CocAction('runCommand', 'editor.action.organizeImport')

" Prettier
command! -nargs=0 Prettier :call CocAction('runCommand', 'prettier.formatFile')

" Add (Neo)Vim's native statusline support.
" NOTE: Please see `:h coc-status` for integrations with external plugins that
" provide custom statusline: lightline.vim, vim-airline.
set statusline^=%{coc#status()}%{get(b:,'coc_current_function','')}

" Mappings using CoCList:
" Show all diagnostics.
nnoremap <silent> <space>a  :<C-u>CocList diagnostics<cr>
" Manage extensions.
nnoremap <silent> <space>x :<C-u>CocList extensions<cr>
" Show commands.
nnoremap <silent> <space>c  :<C-u>CocList commands<cr>
" Find symbol of current document.
nnoremap <silent> <space>o  :<C-u>CocList outline<cr>
" Search workspace symbols.
nnoremap <silent> <space>s  :<C-u>CocList -I symbols<cr>
" Do default action for next item.
nnoremap <silent> <space>j  :<C-u>CocNext<CR>
" Do default action for previous item.
nnoremap <silent> <space>k  :<C-u>CocPrev<CR>
" Resume latest coc list.
nnoremap <silent> <space>p  :<C-u>CocListResume<CR>

" jump to newer location
:unmap <C-i>

" close buffer without closing the window
command! Bd bp|bd #

let g:lightline = {
  \   'colorscheme': 'palenight',
  \   'active': {
  \     'left':[ [ 'mode', 'paste' ],
  \              [ 'gitbranch', 'readonly', 'absolutepath', 'modified' ]
  \     ]
  \   },
    \   'component': {
    \     'lineinfo': ' %3l:%-2v',
    \   },
  \   'component_function': {
  \     'gitbranch': 'fugitive#head',
  \   }
  \ }
let g:lightline.separator = {
    \   'left': '', 'right': ''
  \}
let g:lightline.subseparator = {
    \   'left': '', 'right': ''
  \}
let g:lightline.tabline_separator = {
    \   'left': '', 'right': ''
  \}

let g:vue_disable_pre_processors=1
autocmd FileType vue syntax sync fromstart
autocmd BufReadPost,BufNewFile *.vue setlocal filetype=vue

let NERDTreeHighlightCursorline = 0
let g:NERDTreeFileExtensionHighlightFullName = 1
let g:NERDTreeExactMatchHighlightFullName = 1
let g:NERDTreePatternMatchHighlightFullName = 1

filetype plugin on
nmap gc <plug>NERDCommenterToggle
vmap gc <plug>NERDCommenterToggle

autocmd FileType markdown let g:indentLine_enabled=0
