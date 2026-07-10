-- manifest.lua — load + query .tools/manifest.json (the committed source of
-- truth, shipped inside the plugin clone). All backend hooks go through here.

local json = require("json")
local config = require("config")
local util = require("util")

local M = {}

-- Resolve the manifest path from the plugin dir. RUNTIME.pluginDirPath points
-- at the cloned plugin root (.tools/ after the subdir install).
function M.path()
    return RUNTIME.pluginDirPath .. "/manifest.json"
end

local cached

--- Load + cache the manifest. Errors propagate (caller wraps with context).
function M.load()
    if cached then return cached end
    local f, err = io.open(M.path(), "r")
    if not f then
        error("tools: manifest.json not found at " .. M.path()
            .. " (is the plugin installed from the .tools subdir? " .. tostring(err) .. ")")
    end
    local body = f:read("*a")
    f:close()
    local ok, data = pcall(json.decode, body)
    if not ok then
        error("tools: manifest.json is invalid JSON: " .. tostring(data))
    end
    cached = data
    return data
end

--- Find the manifest cell for a fully-resolved (tool, version, buildgo, os, arch).
--- Returns the cell table or nil.
function M.find_cell(man, tool, version, buildgo, goos, goarch)
    for _, cell in ipairs(man.cells or {}) do
        if cell.tool == tool
            and cell.version == version
            and (cell.buildgo or "") == (buildgo or "")
            and cell.os == goos
            and cell.arch == goarch then
            return cell
        end
    end
    return nil
end

--- Derive the list of selectable version strings for a tool from the manifest.
--- Emits clean keys as primaries (so @latest / semver sort stay stable) plus
--- the opt-in full-suffix forms. Deduped; order does not matter (mise sorts).
function M.versions_for(man, tool)
    local seen = {}
    local out = {}
    local function add(v)
        if v and not seen[v] then seen[v] = true; out[#out + 1] = v end
    end

    for _, cell in ipairs(man.cells or {}) do
        if cell.tool == tool then
            -- clean key (e.g. "1.26", "2.1")
            add(cell.version)
            if tool == "golangci-lint" then
                -- build-go-suffixed key (e.g. "2.1-go1.26")
                add(cell.version .. "-go" .. cell.buildgo)
            end
            -- fully pinned cell (e.g. "1.26-darwin-arm64", "2.1-go1.26-darwin-arm64")
            local full = cell.version
            if tool == "golangci-lint" then
                full = full .. "-go" .. cell.buildgo
            end
            full = full .. "-" .. cell.os .. "-" .. cell.arch
            add(full)
        end
    end
    return out
end

--- Default build-go for golangci-lint, preferring the manifest field.
function M.default_buildgo(man)
    return man.default_buildgo or config.default_buildgo
end

return M
