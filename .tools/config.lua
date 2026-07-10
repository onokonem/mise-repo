-- config.lua
-- Static provider configuration baked into the plugin. Read by the hooks via
-- require("config"). `release_base_url` is the GitHub Releases download root;
-- it is the FALLBACK used to construct asset_url when a manifest cell omits
-- asset_url (it normally does not — manifest-gen fills asset_url explicitly —
-- but the fallback keeps a hand-edited manifest installable).
--
-- The authoritative release tag lives in the committed .tools/manifest.json
-- (release_tag field). asset_url for a cell is always base + release_tag + asset.

local M = {}

-- Owner/repo of the provider. Bump if the repo moves.
M.owner = "onokonem"
M.repo = "mise-repo"

-- Release download base, WITHOUT the tag. A full asset URL is:
--   M.release_base_url .. "/" .. manifest.release_tag .. "/" .. cell.asset
M.release_base_url = "https://github.com/" .. M.owner .. "/" .. M.repo .. "/releases/download"

-- Clean build-go applied when a built-tool version omits -go<buildgo>.
-- Mirrors manifest.default_buildgo; this is only the fallback if the manifest
-- lacks the field.
M.default_buildgo = "1.26"

-- Supported matrix axes. Kept here so the hooks can validate a parsed suffix
-- against the published set even before consulting the manifest.
M.go_versions = { "1.25", "1.26", "1.27-beta" }
M.oses = { "linux", "darwin" }
M.arches = { "amd64", "arm64" }

-- Filename conventions (informational; the manifest carries asset_url for every
-- cell, so these are only fallbacks for a hand-edited manifest). go mirrors the
-- official tarball name; every built tool is tarballed as <binary>-<stem>-go<buildgo>.<os>-<arch>.tar.gz.
function M.go_asset_name(resolved_tag, goos, goarch)
    return resolved_tag .. "." .. goos .. "-" .. goarch .. ".tar.gz"
end

function M.built_asset_name(binary, version, buildgo, goos, goarch)
    local stem = version:gsub("^v", "")
    return binary .. "-" .. stem .. "-go" .. buildgo .. "." .. goos .. "-" .. goarch .. ".tar.gz"
end

return M
