vim.g.loaded_netrwPlugin = 1
vim.g.loaded_netrw = 1

local yank = require('custom.yank')
local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not vim.loop.fs_stat(lazypath) then
  vim.fn.system({
    "git",
    "clone",
    "--filter=blob:none",
    "https://github.com/folke/lazy.nvim.git",
    "--branch=stable", -- latest stable release
    lazypath,
  })
end
vim.opt.rtp:prepend(lazypath)


require('lazy').setup({
  { 'Exafunction/codeium.vim' },
  -- color theme
  "ellisonleao/gruvbox.nvim",
  { "chentoast/marks.nvim",         event = "VeryLazy", opts = {}, },
  { "echasnovski/mini.ai",          version = "*" },
  { "echasnovski/mini.basics",      version = "*" },
  { "echasnovski/mini.bracketed",   version = "*" },
  { "echasnovski/mini.bufremove",   version = "*" },
  { "echasnovski/mini.comment",     version = "*" },
  { 'echasnovski/mini.extra',       version = '*' },
  { 'echasnovski/mini.files',       version = '*' },
  { 'echasnovski/mini.icons',       version = '*' },
  { 'echasnovski/mini.indentscope', version = '*' },
  { 'echasnovski/mini.jump',        version = '*' },
  { 'echasnovski/mini.jump2d',      version = '*' },
  { "echasnovski/mini.move",        version = "*" },
  { "echasnovski/mini.nvim",        version = "*" },
  { "echasnovski/mini.operators",   version = "*" },
  { "echasnovski/mini.pick",        version = "*" },
  { "echasnovski/mini.pairs",       version = "*" },
  { 'echasnovski/mini.snippets',    version = '*' },
  { 'echasnovski/mini.nvim',        version = '*' },
  { "echasnovski/mini.statusline",  version = "*" },
  { "echasnovski/mini.starter",     version = "*" },
  { "echasnovski/mini.surround",    version = "*" },
  { 'echasnovski/mini.trailspace',  version = '*' },
  { 'echasnovski/mini.visits',      version = '*' },

  { 'sainnhe/gruvbox-material' },

  { 'tpope/vim-fugitive' },
  {
    "nvim-neo-tree/neo-tree.nvim",
    branch = "v3.x",
    dependencies = {
      "nvim-lua/plenary.nvim",
      "nvim-tree/nvim-web-devicons",
      "MunifTanjim/nui.nvim",
    },
    config = function()
      require("neo-tree").setup({
        window = {
          auto_expand_width = true
        }
      })
    end
  },
  {
    'kevinhwang91/nvim-ufo',
    config = function()
      require('ufo').setup({
        provider_selector = function(bufnr, filetype, buftype)
          return { 'treesitter', 'indent' }
        end
      })
    end,
    dependencies = 'kevinhwang91/promise-async'
  },
  -- TreeSitter
  'nvim-treesitter/nvim-treesitter',
  'nvim-treesitter/nvim-treesitter-context',
  'HiPhish/nvim-ts-rainbow2',
  -- formatting & linting
  {
    'nvimtools/none-ls.nvim',
    dependencies = {
      "nvimtools/none-ls-extras.nvim",
    },
  },
  { 'neovim/nvim-lspconfig' },
  'williamboman/mason.nvim',
  'williamboman/mason-lspconfig.nvim',

  -- Autocompletion
  'hrsh7th/nvim-cmp',
  'hrsh7th/cmp-nvim-lsp',
  'hrsh7th/cmp-path',
  { 'L3MON4D3/LuaSnip',     version = "v2.*" },
  'saadparwaiz1/cmp_luasnip',

  {
    'VonHeikemen/lsp-zero.nvim',
    branch = 'v4.x',
  },

  "rafamadriz/friendly-snippets",
  -- git
  'lewis6991/gitsigns.nvim',
  -- typescript
  'jose-elias-alvarez/typescript.nvim',
  -- auto closing
  'windwp/nvim-ts-autotag',

  'editorconfig/editorconfig-vim',
  'cappyzawa/trim.nvim',

  -- colorizer
  'norcalli/nvim-colorizer.lua',

  "nvim-lua/plenary.nvim",

  'ray-x/lsp_signature.nvim',
  {
    'stevearc/conform.nvim',
    opts = {},
  }
})

-- --------------------
-- Global configuration
-- --------------------

local keymap = vim.keymap
local opt = vim.opt
local cmd = vim.cmd

vim.o.directory = "/tmp"

-- nvim-ufo
vim.o.foldcolumn = '1' -- '0' is not bad
vim.o.foldlevel = 99   -- Using ufo provider need a large value, feel free to decrease the value
vim.o.foldlevelstart = 99
vim.o.foldenable = true

vim.o.undofile = false

vim.g.mapleader = " "
cmd("colorscheme gruvbox-material")

-- see :help E355
opt.list = true
opt.updatetime = 750

opt.guicursor = ''
opt.mouse = ''
opt.hlsearch = false
opt.incsearch = true

-- line numbers
opt.relativenumber = true
opt.number = true

-- tabs & indentation
opt.tabstop = 2
opt.shiftwidth = 2
opt.expandtab = true
opt.autoindent = true

-- line wrapping
opt.wrap = false

-- search settings
opt.ignorecase = true
opt.smartcase = true

-- cursor line
opt.cursorline = true

-- appearance
opt.termguicolors = true
opt.background = "dark"
opt.signcolumn = "yes"

-- backspace
opt.backspace = "indent,eol,start"

-- clipboard
opt.clipboard:append("unnamedplus")


-- split windows
opt.splitright = true
opt.splitbelow = true


-- keywords
opt.iskeyword:append("-")

keymap.set("n", "n", "nzzzv")
keymap.set("n", "N", "Nzzzv")
keymap.set("x", "<leader>p", "\"_dP")

keymap.set("i", "jj", "<ESC>")           -- close insert mode
keymap.set("n", "<space>l", ":nohl<CR>") -- clear search results
keymap.set("n", ";", ":")

keymap.set("n", "<leader>+", "<C-a>")          -- increment
keymap.set("n", "<leader>-", "<C-x>")          -- decrement

keymap.set("n", "<leader>sv", "<C-w>v")        -- split window vertically
keymap.set("n", "<leader>sh", "<C-w>s")        -- split window horizontally
keymap.set("n", "<leader>se", "<C-w>=")        -- make split windows equal width
keymap.set("n", "<leader>sx", ":close<CR>")    -- close current split window

keymap.set("n", "<leader>to", ":tabnew<CR>")   -- open new tab
keymap.set("n", "<leader>tx", ":tabclose<CR>") -- close current tab
keymap.set("n", "<leader>tn", ":tabn<CR>")     -- go to next tab
keymap.set("n", "<leader>tp", ":tabp<CR>")     -- go to next tab

keymap.set('n', '<leader>ld', vim.diagnostic.open_float)
keymap.set('n', '[d', vim.diagnostic.goto_prev)
keymap.set('n', ']d', vim.diagnostic.goto_next)
keymap.set('n', '<space>q', vim.diagnostic.setloclist)

-- -------
-- Plugins
-- -------
require("mini.extra").setup() -- has to come before mini.pick
local mini_pick = require('mini.pick')
vim.ui.select = mini_pick.ui_select

-- load vs-code like snippets from plugins (e.g. friendly-snippets)
-- require("luasnip/loaders/from_vscode").lazy_load()

vim.opt.completeopt = "menu,menuone,noselect"

-- TreeSitter
require("nvim-treesitter.configs").setup({
  ensure_installed = {
    "json",
    "javascript",
    "typescript",
    "tsx",
    "yaml",
    "html",
    "css",
    "markdown",
    "markdown_inline",
    "bash",
    "lua",
    "vim",
    "dockerfile",
    "gitignore",
    "git_rebase",
    "gitcommit"
  },
  sync_install = false,
  auto_install = true,
  -- enable autotagging (w/ nvim-ts-autotag plugin)
  autotag = { enable = true },
  higlight = {
    enable = true
  },
  indent = {
    enable = true,
    disable = {},
  },
})

--gitsigns
require('gitsigns').setup({
  on_attach = function(bufnr)
    local gs = package.loaded.gitsigns

    local function map(mode, l, r, opts)
      opts = opts or {}
      opts.buffer = bufnr
      vim.keymap.set(mode, l, r, opts)
    end

    -- Navigation
    map('n', ']c', function()
      if vim.wo.diff then return ']c' end
      vim.schedule(function() gs.next_hunk() end)
      return '<Ignore>'
    end, { expr = true })

    map('n', '[c', function()
      if vim.wo.diff then return '[c' end
      vim.schedule(function() gs.prev_hunk() end)
      return '<Ignore>'
    end, { expr = true })

    map({ 'n', 'v' }, '<leader>hs', ':Gitsigns stage_hunk<CR>')
    map({ 'n', 'v' }, '<leader>hr', ':Gitsigns reset_hunk<CR>')
    map('n', '<leader>hu', gs.undo_stage_hunk)
  end
})

-- colorizer
require('colorizer').setup()
--

local lspconfig_defaults = require('lspconfig').util.default_config
lspconfig_defaults.capabilities = vim.tbl_deep_extend(
  'force',
  lspconfig_defaults.capabilities,
  require('cmp_nvim_lsp').default_capabilities()
)

-- This is where you enable features that only work
-- if there is a language server active in the file
vim.api.nvim_create_autocmd('LspAttach', {
  desc = 'LSP actions',
  callback = function(event)
    local opts = { buffer = event.buf }

    vim.keymap.set('n', 'K', '<cmd>lua vim.lsp.buf.hover()<cr>', opts)
    vim.keymap.set('n', 'gd', '<cmd>lua MiniExtra.pickers.lsp({ scope = "definition"})<cr>', opts)
    vim.keymap.set('n', 'gD', '<cmd>lua MiniExtra.pickers.lsp({ scope = "declaration"})<cr>', opts)
    vim.keymap.set('n', 'gi', '<cmd>lua MiniExtra.pickers.lsp({ scope = "implemenation"})<cr>', opts)
    vim.keymap.set('n', 'go', '<cmd>lua MiniExtra.pickers.lsp({ scope = "type_definition"})<cr>', opts)
    vim.keymap.set('n', 'gr', '<cmd>lua MiniExtra.pickers.lsp({ scope = "references"})<cr>', opts)
    vim.keymap.set('n', 'gs', '<cmd>lua vim.lsp.buf.signature_help()<cr>', opts)
    vim.keymap.set('n', '<space>ca', '<cmd>lua vim.lsp.buf.code_action()<cr>', opts)
    vim.keymap.set('n', '<space>rn', '<cmd>lua vim.lsp.buf.rename()<cr>', opts)
  end,
})

require('mason').setup({})
require('mason-lspconfig').setup({
  ensure_installed = {},
  handlers = {
    function(server_name)
      require('lspconfig')[server_name].setup({})
    end,
  }
})

local cmp = require('cmp')
local lsp_zero = require('lsp-zero')

cmp.setup({
  sources = {
    { name = "cody" },
    { name = 'nvim_lsp' },
    { name = 'luasnip' },
  },
  snippet = {
    expand = function(args)
      -- You need Neovim v0.10 to use vim.snippet
      vim.snippet.expand(args.body)
    end,
  },
  mapping = cmp.mapping.preset.insert({
    ["<C-k>"] = cmp.mapping.select_prev_item(), -- previous suggestion
    ["<C-j>"] = cmp.mapping.select_next_item(), -- next suggestion
    ["<C-b>"] = cmp.mapping.scroll_docs(-4),
    ["<C-f>"] = cmp.mapping.scroll_docs(4),
    ["<C-Space>"] = cmp.mapping.complete(), -- show completion suggestions
    ["<C-e>"] = cmp.mapping.abort(),        -- close completion window
    ["<CR>"] = cmp.mapping.confirm({ select = false }),
  }),
})

lsp_zero.set_sign_icons({
  error = '✘',
  warn = '▲',
  hint = '⚑',
  info = '»'
})
--]]

local cmp = require('cmp')

local setup, null_ls = pcall(require, "null-ls")
if not setup then
  return
end

-- for conciseness
local formatting = null_ls.builtins.formatting   -- to setup formatters
local diagnostics = null_ls.builtins.diagnostics -- to setup linters

-- to setup format on save
local augroup = vim.api.nvim_create_augroup("LspFormatting", {})

null_ls.setup({
  debug = true,
  sources = {
    null_ls.builtins.code_actions.gitsigns,
    require("none-ls.code_actions.eslint_d"),
    require("none-ls.diagnostics.eslint_d"),
    formatting.prettierd,
  },
  on_attach = function(current_client, bufnr)
    if current_client.supports_method("textDocument/formatting") then
      vim.api.nvim_clear_autocmds({ group = augroup, buffer = bufnr })
    end
  end,
})

vim.diagnostic.config({
  virtual_text = false,
  severity_sort = true,
  float = {
    border = 'rounded',
    source = 'always',
  },
})

-- lsp_signature
local signature_config = {
  log_path = "/tmp/sig.log",
  debug = true,
  hint_enable = false,
  handler_opts = { border = "single" },
  max_width = 80,
}
require("lsp_signature").setup(signature_config)

-- fugitive
vim.keymap.set('n', "<leader>gg", ":G<CR>")

-- treesitter
require 'treesitter-context'.setup {
  max_lines = 1
}

-- conform
require("conform").setup({
  formatters_by_ft = {
    lua = { "stylua" },
    javascript = { "prettierd", "eslint_d" },
    typescript = { "prettierd", "eslint_d" },
    javascriptreact = { "prettierd", "eslint_d" },
    typescriptreact = { "prettierd", "eslint_d" },
  },
  format_after_save = {
    -- These options will be passed to conform.format()
    async = true,
    lsp_format = "fallback"
  }
})

-- mini
require("mini.ai").setup({
  n_lines = 200
})
require("mini.basics").setup({
  mappings = {
    windows = true
  }
})
require("mini.bracketed").setup()
require("mini.bufremove").setup()
require("mini.comment").setup()
require("mini.extra").setup()
require("mini.files").setup({
  mappings = {
    trim_left = '',
    trim_right = '',
    go_out = ',',
    go_in = '.',
    go_out_plus = "<",
    go_in_plus = ">",

  }
})
require('mini.icons').setup()
require("mini.indentscope").setup({
  draw = {
    animation = require('mini.indentscope').gen_animation.none()
  }
})

local jump2d = require("mini.jump2d")
jump2d.setup({
  spotter = jump2d.builtin_opts.word_start.spotter,
  view = {
    dim = true
  },
  mappings = {
    start_jumping = '<leader><leader>w'
  }
})
require("mini.jump").setup({
  mappings = {
    repeat_jump = ''
  },
  delay = {
    highlight = 10000000
  }
})
require("mini.move").setup({
  mappings = {
    left = '',
    right = '',
    down = '<leader>j',
    up = '<leader>k',

    line_left = '',
    line_right = '',
    line_down = ']e',
    line_up = '[e'
  }
})
require("mini.operators").setup({
  replace = {
    prefix = ''
  }
})
require("mini.pairs").setup()
mini_pick.setup()
require("mini.statusline").setup()
require("mini.surround").setup({
  custom_surroundings = {
    ['"'] = { input = { '%b""', '^.().*().$' } },
  },
  search_method = 'cover_or_next',
})
require("mini.trailspace").setup()
require("mini.visits").setup()

-- mini.visits
keymap.set("n", "<leader>vl", "<cmd>lua MiniVisits.select_path()<cr>")
keymap.set("n", "<leader>ll", "<cmd>lua MiniVisits.select_label()<cr>")
keymap.set("n", "<leader>la", "<cmd>lua MiniVisits.add_label()<cr>")

-- mini.pick
keymap.set("n", "<leader>f;", "<cmd>Pick files tool='git'<cr>")
keymap.set("n", "<leader>fd", "<cmd>Pick diagnostic scope='all'<cr>")
keymap.set("n", "<leader>fD", "<cmd>Pick diagnostic scope='current'<cr>")
keymap.set("n", "<leader>fh", "<cmd>Pick visit_paths<cr>")            -- find recent files
keymap.set("n", "<leader>fs", "<cmd>Pick grep_live<cr>")              -- find string in current working directory as you type
keymap.set("n", "<leader>fc", "<cmd>Pick grep pattern='<cword>'<cr>") -- find string in current working directory as you type
keymap.set("n", "<leader>fb", "<cmd>Pick buffers<cr>")                -- list open buffers in current neovim instance
keymap.set("n", "<leader>ch", "<cmd>Pick history scope='cmd'<cr>")    -- list current changes per file with diff preview ["gs" for git status]
keymap.set("n", "<leader>sh", "<cmd>Pick history scope='search'<cr>") -- list current changes per file with diff preview ["gs" for git status]
keymap.set("n", "<leader>tr", "<cmd>Pick resume<cr>")                 -- resume last picker
keymap.set('n', '<leader>yr', function()
  yank.yank_path(yank.get_buffer_cwd_relative(), 'relative')
end, { desc = '[Y]ank [R]elative file path' })
keymap.set('n', '<leader>ya', function()
  yank.yank_path(yank.get_buffer_absolute(), 'absolute')
end, { desc = '[Y]ank [A]bsolute file path' })
keymap.set('v', '<leader>yr', function()
  yank.yank_visual_with_path(yank.get_buffer_cwd_relative(), 'relative')
end, { desc = '[Y]ank selection with [R]elative path' })


-- mini.bufremove
keymap.set("n", "<leader>bd", "<cmd>lua MiniBufremove.delete()<cr>")

-- mini.files
keymap.set("n", "<leader>e", function()
  local MiniFiles = require("mini.files")
  local _ = MiniFiles.close()
      or MiniFiles.open(vim.api.nvim_buf_get_name(0), false)
end
)

keymap.set("n", "<leader>n", function()
  local MiniFiles = require("mini.files")
  local _ = MiniFiles.close()
      or MiniFiles.open(vim.api.nvim_buf_get_name(0), false)
  vim.schedule(function()
    MiniFiles.reveal_cwd()
  end, 30)
end)

-- mini.snippets
local javascript_patterns = { '**/react.json' }
local lang_patterns = { tsx = javascript_patterns, typescript = javascript_patterns }
local gen_loader = require('mini.snippets').gen_loader
require('mini.snippets').setup({
  snippets = {
    gen_loader.from_file('~/.config/nvim/snippets/global.json'),
    gen_loader.from_lang({ lang_patterns = lang_patterns }),
  },
})

-- mini.starter
local starter = require('mini.starter')
starter.setup({
  items = {
    starter.sections.builtin_actions(),
    starter.sections.recent_files(10, false),
    starter.sections.recent_files(10, true),
  }
})

-- custom
require('custom.directory-watcher').setup({ path = vim.fn.getcwd() })
require('custom.hotreload').setup()
