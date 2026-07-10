--- BackendListVersions — advertise selectable versions for one tool.
--- Backend (multi-tool) equivalent of the single-tool Available hook. mise calls
--- this per tool via the plugin:tool address; ctx.tool is "go" or "golangci-lint".
--- @param ctx {tool: string, options: table}
--- @return {versions: string[]}
function PLUGIN:BackendListVersions(ctx)
    local manifest = require("manifest")
    local man = manifest.load()
    return { versions = manifest.versions_for(man, ctx.tool) }
end
