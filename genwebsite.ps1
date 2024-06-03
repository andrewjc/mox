#!/usr/bin/env pwsh

# Create directory if it doesn't exist
New-Item -ItemType Directory -Force -Path website/html

# Remove all files in the directory
Remove-Item -Recurse -Force website/html/*

# Exit on error
$ErrorActionPreference = "Stop"

# Get commit hash and date
$commithash = git rev-parse --short HEAD
$commitdate = git log -1 --date=format:"yyyy-MM-dd" --format="%ad"
$env:commithash = $commithash
$env:commitdate = $commitdate

# Link to static files and cross-references
New-Item -ItemType SymbolicLink -Path website/html/files -Target ../../../mox-website-files/files -Force
New-Item -ItemType SymbolicLink -Path website/html/xr -Target ../../rfc/xr -Force

# Change to the website directory
Set-Location -Path website

# Generate HTML files
go run website.go -root -title 'Mox: modern, secure, all-in-one mail server' 'Mox' < index.md > html/index.html

# Generate features HTML
New-Item -ItemType Directory -Path html/features -Force
(
    Get-Content features/index.md
    Write-Output ""
    (Get-Content ../README.md | Select-String -Pattern '# FAQ' -NotMatch | Select-String -Pattern '## Roadmap' -Context 0,1000)
    Write-Output ""
    Write-Output 'Also see the [Protocols](../protocols/) page for implementation status, and (non)-plans.'
) | go run website.go 'Features' > html/features/index.html

# Generate screenshots HTML
New-Item -ItemType Directory -Path html/screenshots -Force
go run website.go 'Screenshots' < screenshots/index.md > html/screenshots/index.html

# Generate install HTML
New-Item -ItemType Directory -Path html/install -Force
go run website.go 'Install' < install/index.md > html/install/index.html

# Generate FAQ HTML
New-Item -ItemType Directory -Path html/faq -Force
(Get-Content ../README.md | Select-String -Pattern '# FAQ' -Context 0,1000) | go run website.go 'FAQ' > html/faq/index.html

# Generate config reference HTML
New-Item -ItemType Directory -Path html/config -Force
(
    Write-Output '# Config reference'
    Write-Output ""
    (Get-Content ../config/doc.go | Select-String -Pattern '^Package config holds ', '^\*/' -NotMatch) -replace '^# ', '## '
) | go run website.go 'Config reference' > html/config/index.html

# Generate command reference HTML
New-Item -ItemType Directory -Path html/commands -Force
(
    Write-Output '# Command reference'
    Write-Output ""
    (Get-Content ../doc.go | Select-String -Pattern '^Mox is started ', '^\*/' -NotMatch) -replace '^# ', '## '
) | go run website.go 'Command reference' > html/commands/index.html

# Generate protocols HTML
New-Item -ItemType Directory -Path html/protocols -Force
go run website.go -protocols 'Protocols' < ../rfc/index.txt > html/protocols/index.html

# Create HTML file for the build page
New-Item -ItemType Directory -Path html/b -Force
@"
<!doctype html>
<html>
    <head>
        <meta charset="utf-8" />
        <title>mox build</title>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="noNeedlessFaviconRequestsPlease:" />
        <style>
body { padding: 1em; }
* { font-size: 18px; font-family: ubuntu, lato, sans-serif; margin: 0; padding: 0; box-sizing: border-box; }
p { max-width: 50em; margin-bottom: 2ex; }
pre { font-family: 'ubuntu mono', monospace; }
pre, blockquote { padding: 1em; background-color: #eee; border-radius: .25em; display: inline-block; margin-bottom: 1em; }
h1 { margin: 1em 0 .5em 0; }
        </style>
    </head>
    <body>
<script>
const elem = (name, ...s) => {
    const e = document.createElement(name)
    e.append(...s)
    return e
}
const link = (url, anchor) => {
    const e = document.createElement('a')
    e.setAttribute('href', url)
    e.setAttribute('rel', 'noopener')
    e.append(anchor || url)
    return e
}
let h = location.hash.substring(1)
const ok = /^[a-zA-Z0-9_\.]+$/.test(h)
if (!ok) {
    h = '<tag-or-branch-or-commithash>'
}
const init = () => {
    document.body.append(
        elem('p', 'Compile or download any version of mox, by tag (release), branch or commit hash.'),
        elem('h1', 'Compile'),
        elem('p', 'Run:'),
        elem('pre', 'CGO_ENABLED=0 GOBIN=$PWD go install github.com/mjl-/mox@'+h),
        elem('p', 'Mox is tested with the Go toolchain versions that are still have support: The most recent version, and the version before.'),
        elem('h1', 'Download'),
        elem('p', 'Download a binary for your platform:'),
        elem('blockquote', ok ?
            link('https://beta.gobuilds.org/github.com/mjl-/mox@'+h) :
            'https://beta.gobuilds.org/github.com/mjl-/mox@'+h
        ),
        elem('p', 'Because mox is written in Go, builds are reproducible, also when cross-compiling. Gobuilds.org is a service that builds Go applications on-demand with the latest Go toolchain/runtime.'),
        elem('h1', 'Localserve'),
        elem('p', 'Changes to mox can often be most easily tested locally with ', link('../features/#hdr-localserve', '"mox localserve"'), ', without having to update your running mail server.'),
    )
}
window.addEventListener('load', init)
</script>
    </body>
</html>
"@ | Out-File -FilePath html/b/index.html
