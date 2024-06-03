# ./doc.go
@"
/*
Command mox is a modern, secure, full-featured, open source mail server for
low-maintenance self-hosted email.

Mox is started with the "serve" subcommand, but mox also has many other
subcommands.

Many of those commands talk to a running mox instance, through the ctl file in
the data directory. Specify the configuration file (that holds the path to the
data directory) through the -config flag or MOXCONF environment variable.

Commands that don't talk to a running mox instance are often for
testing/debugging email functionality. For example for parsing an email message,
or looking up SPF/DKIM/DMARC records.

Below is the usage information as printed by the command when started without
any parameters. Followed by the help and usage information for each command.


# Usage

"@ | Out-File -FilePath doc.go

& ./mox.exe 2>&1 | ForEach-Object {$_ -replace '^usage: *', "`t" -replace '^  *', "`t"} | Out-File -Append -FilePath doc.go
"`n" | Out-File -Append -FilePath doc.go

& ./mox.exe helpall 2>&1 | Out-File -Append -FilePath doc.go

@"
*/
package main

// NOTE: DO NOT EDIT, this file is generated by gendoc.ps1.

"@ | Out-File -Append -FilePath doc.go

& gofmt -w doc.go

# ./config/doc.go
@"
/*
Package config holds the configuration file definitions.

Mox uses two config files:

1. mox.conf, also called the static configuration file.
2. domains.conf, also called the dynamic configuration file.

The static configuration file is never reloaded during the lifetime of a
running mox instance. After changes to mox.conf, mox must be restarted for the
changes to take effect.

The dynamic configuration file is reloaded automatically when it changes.
If the file contains an error after the change, the reload is aborted and the
previous version remains active.

Below are "empty" config files, generated from the config file definitions in
the source code, along with comments explaining the fields. Fields named "x" are
placeholders for user-chosen map keys.

# sconf

The config files are in "sconf" format. Properties of sconf files:

- Indentation with tabs only.
- "#" as first non-whitespace character makes the line a comment. Lines with a
  value cannot also have a comment.
- Values don't have syntax indicating their type. For example, strings are
  not quoted/escaped and can never span multiple lines.
- Fields that are optional can be left out completely. But the value of an
  optional field may itself have required fields.

See https://pkg.go.dev/github.com/mjl-/sconf for details.


# mox.conf


"@ | Out-File -FilePath config/doc.go

& ./mox.exe config describe-static | ForEach-Object {"`t$_"} | Out-File -Append -FilePath config/doc.go

@"
# domains.conf

"@ | Out-File -Append -FilePath config/doc.go

& ./mox.exe config describe-domains | ForEach-Object {"`t$_"} | Out-File -Append -FilePath config/doc.go

@"
# Examples

Mox includes configuration files to illustrate common setups. You can see these
examples with "mox config example", and print a specific example with "mox
config example <name>". Below are all examples included in mox.

"@ | Out-File -Append -FilePath config/doc.go

foreach ($ex in & ./mox.exe config example) {
    "# Example $ex`n" | Out-File -Append -FilePath config/doc.go
    & ./mox config example $ex | ForEach-Object {"`t$_"} | Out-File -Append -FilePath config/doc.go
    "`n" | Out-File -Append -FilePath config/doc.go
}

@"
*/
package config

// NOTE: DO NOT EDIT, this file is generated by ../gendoc.ps1.
"@ | Out-File -Append -FilePath config/doc.go

& gofmt -w config/doc.go

# ./webapi/doc.go
& ./webapi/gendoc.ps1 | Out-File -FilePath webapi/doc.go
& gofmt -w webapi/doc.go