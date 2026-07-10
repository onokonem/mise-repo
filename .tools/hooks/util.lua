-- util.lua — shared helpers for the tools backend hooks.
-- Pure functions: no network, no fs writes. Safe to require from every hook.

local config = require("config")

local M = {}

--- Host (goos, goarch) derived from the vfox RUNTIME, mapped to Go conventions.
--- RUNTIME.osType is "Darwin"/"Linux"/"Windows"; archType is "amd64"/"arm64"/"386".
function M.host()
    local goos = RUNTIME.osType:lower()
    if goos == "macos" then goos = "darwin" end
    local goarch = RUNTIME.archType
    if goarch == "x86_64" then goarch = "amd64" elseif goarch == "aarch64" then goarch = "arm64" end
    return goos, goarch
end

--- Parse a requested version string into its selection components.
--- Grammar: <ver>[-go<buildgo>][-<goos>-<goarch>]
---   "1.26"                         -> {version="1.26"}
---   "1.27-beta-darwin-arm64"       -> {version="1.27-beta", goos="darwin", goarch="arm64"}
---   "2.1-go1.26"                   -> {version="2.1", buildgo="1.26"}
---   "2.1-go1.27-beta-linux-amd64"  -> {version="2.1", buildgo="1.27-beta", goos="linux", goarch="amd64"}
--- Returns a table always carrying version; goos/goarch/buildgo are nil when absent.
function M.parse(version)
    local parsed = { version = version, buildgo = nil, goos = nil, goarch = nil }
    local rest = version

    -- 1. strip a trailing -<goos>-<goarch> if it matches a known axis cell.
    local found_cell = false
    for _, goos in ipairs(config.oses) do
        for _, goarch in ipairs(config.arches) do
            local suf = "-" .. goos .. "-" .. goarch
            if rest:sub(-#suf) == suf then
                parsed.goos, parsed.goarch = goos, goarch
                rest = rest:sub(1, #rest - #suf)
                found_cell = true
                break
            end
        end
        if found_cell then break end
    end

    -- 2. strip a trailing -go<buildgo> (only present for golangci-lint).
    --    buildgo starts with a digit: "1.26", "1.27-beta". The "-go" must be
    --    literal so a plain Go version like "1.27-beta" is never mis-split.
    local prefix, bg = rest:match("^(.*)%-go([0-9][%w.-]*)$")
    if prefix then
        parsed.buildgo = bg
        rest = prefix
    end
    parsed.version = rest
    return parsed
end

--- Resolve a parsed request to a concrete (tool, version, buildgo, goos, goarch)
--- against the host + manifest defaults. Does NOT consult the manifest; only
--- fills in host/default slots the caller asked to defer.
function M.resolve(parsed, default_buildgo)
    local goos = parsed.goos
    local goarch = parsed.goarch
    if not goos or not goarch then
        local hgoos, hgoarch = M.host()
        goos = goos or hgoos
        goarch = goarch or hgoarch
    end
    local buildgo = parsed.buildgo or default_buildgo
    return {
        version = parsed.version,
        buildgo = buildgo,
        goos = goos,
        goarch = goarch,
    }
end

--- Build the manifest lookup key for a resolved cell. Matches the cell table
--- shape stored in manifest.json.
function M.cell_key(tool, version, buildgo, goos, goarch)
    return tool .. "|" .. version .. "|" .. (buildgo or "") .. "|" .. goos .. "|" .. goarch
end

return M
