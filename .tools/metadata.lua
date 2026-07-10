-- metadata.lua
-- `tools`: a single mise vfox *backend* (multi-tool) plugin serving two tools
-- — `go` and `golangci-lint` — addressed as tools:go and tools:golangci-lint.
--
-- mise classifies a plugin as a backend (multi-tool) plugin when BOTH
-- metadata.lua and hooks/backend_install.lua exist at the plugin root
-- (see mise src/plugins/mod.rs from_plugin_path). Backend plugins are invoked
-- via the plugin:tool format and dispatch to BackendListVersions /
-- BackendInstall / BackendExecEnv, each receiving `ctx.tool`. The single-tool
-- Available/PreInstall/EnvKeys hooks are NOT used here.

PLUGIN = {
    name = "tools",
    version = "1.0.0",
    description = "Self-hosted Go + golangci-lint matrix (go x {linux,darwin} x {amd64,arm64} x {1.25,1.26,1.27-beta})",
    author = "onokonem",
    updateUrl = "https://github.com/onokonem/mise-repo",
    minRuntimeVersion = "0.2.0",
}
