--- BackendExecEnv — environment variables exported for an installed tool.
--- Backend (multi-tool) equivalent of the single-tool EnvKeys hook.
--- @param ctx {tool: string, version: string, install_path: string, options: table}
--- @return {env_vars: {key:string,value:string}[]}

function PLUGIN:BackendExecEnv(ctx)
    if ctx.tool == "go" then
        -- Official Go tarball extracts a top-level `go/` dir; GOROOT points at it.
        -- Verify by `go env GOROOT` resolves (task 5.3).
        local goroot = ctx.install_path .. "/go"
        return {
            env_vars = {
                { key = "GOROOT", value = goroot },
                { key = "GOPATH", value = os.getenv("HOME") .. "/go" },
                { key = "PATH", value = goroot .. "/bin" },
            },
        }
    end

    -- golangci-lint: single binary laid out at install_path root. No GOROOT.
    return {
        env_vars = {
            { key = "PATH", value = ctx.install_path },
        },
    }
end
