--- BackendInstall — fetch, verify and lay out one cell for one tool.
--- Backend (multi-tool) equivalent of PreInstall+PostInstall combined. mise
--- downloads nothing for backend plugins; this hook performs the download
--- (http.download_file), verifies sha256 against the manifest, and extracts
--- into ctx.install_path.
--- @param ctx {tool: string, version: string, install_path: string, download_path: string, options: table}
--- @return table

local function shell_quote(s)
    return "'" .. tostring(s):gsub("'", "'\"'\"'") .. "'"
end

-- Compute sha256 of a file using the host's shasum/sha256sum. Returns the hex
-- digest or nil. vfox has no built-in hash module, so shell out.
local function sha256_of(path)
    local cmd = "shasum -a 256 " .. shell_quote(path) .. " 2>/dev/null || sha256sum " .. shell_quote(path) .. " 2>/dev/null"
    local h = io.popen(cmd)
    if not h then return nil end
    local line = h:read("*l")
    h:close()
    if not line then return nil end
    return line:match("^([0-9a-fA-F]+)")
end

function PLUGIN:BackendInstall(ctx)
    local http = require("http")
    local archiver = require("archiver")
    local manifest = require("manifest")
    local util = require("util")

    local man = manifest.load()
    local parsed = util.parse(ctx.version)

    -- Resolve host/default slots.
    local goos, goarch = parsed.goos, parsed.goarch
    if not goos or not goarch then
        local hgoos, hgoarch = util.host()
        goos = goos or hgoos
        goarch = goarch or hgoarch
    end

    -- buildgo: for go it IS the version row; for golangci-lint it is the
    -- explicit -go<buildgo> suffix or the manifest default.
    local buildgo
    if ctx.tool == "go" then
        buildgo = parsed.version
    else
        buildgo = parsed.buildgo or manifest.default_buildgo(man)
    end

    local cell = manifest.find_cell(man, ctx.tool, parsed.version, buildgo, goos, goarch)
    if not cell then
        error(string.format(
            "tools: no published cell for %s@%s (buildgo=%s, %s-%s). " ..
            "The cell was either skipped during build (see publish logs) or the " ..
            "version/suffix is wrong. Run `mise ls tools:%s` to see what is available.",
            ctx.tool, ctx.version, buildgo or "?", goos, goarch, ctx.tool))
    end

    -- Download into the temp download dir mise provisioned.
    local archive = ctx.download_path .. "/" .. cell.asset
    http.download_file({ url = cell.asset_url }, archive)

    -- Byte-identity guard: verify the download matches the manifest sha256.
    local got = sha256_of(archive)
    if not got then
        error("tools: could not compute sha256 of " .. archive .. " (need shasum or sha256sum on PATH)")
    end
    if got:lower() ~= cell.sha256:lower() then
        error("tools: sha256 mismatch for " .. cell.asset ..
              "\n  expected " .. cell.sha256 ..
              "\n  got      " .. got:lower())
    end

    -- Lay out into install_path. mkdir -p via os.execute; archiver needs the dest.
    os.execute("mkdir -p " .. shell_quote(ctx.install_path))
    archiver.decompress(archive, ctx.install_path)

    -- Executable bit is not preserved by every extractor; ensure linter runs.
    if ctx.tool == "golangci-lint" then
        os.execute("chmod +x " .. shell_quote(ctx.install_path .. "/golangci-lint") .. " 2>/dev/null")
    end

    return {}
end
