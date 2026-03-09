# hurl-lsp

A Language Server Protocol (LSP) server for [Hurl](https://hurl.dev/).

`hurl-lsp` provides language server capabilities for Hurl files.
It can optionally integrate with your OpenAPI specifications to provide spec-aware suggestions and validations.

## Installation

At the moment you can only install this by cloning and building yourself.
```bash
git clone git@github.com:ethancarlsson/hurl-lsp.git
cd hurl-lsp
go install
```
## Usage

Configure your LSP client to use the `hurl-lsp` binary.

### Neovim

```lua
vim.lsp.config["hurl_ls"] = {
	cmd = { "hurl-lsp" },
	filetypes = {"hurl"}
}

vim.lsp.enable("hurl_ls")
```

## Configuration

To enable OpenAPI-aware features, create a configuration file named `.hurl-ls.json` in the root of your project.

### Example `.hurl-ls.json`

```json
{
  "openapi_def": [
    "./api-spec.yaml",
    "./another-spec.json"
  ]
}
```

- **`openapi_def`**: An array of paths to your OpenAPI definition files. Both JSON and YAML formats are supported.
