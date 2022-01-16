# AWS SSO CLI Changelog

## [1.7.1] - Unreleased

### Bug Fixes

 * `AWS_SSO` env var is now set with the `eval` and `exec` command #251
 * Fix broken auto-complete for non-Default AWS SSO instances #249
 * Fix incorrect `AWS_SSO_SESSION_EXPIRATION` values #250
 * Remove old config settings that no longer exist #254
 * `cache` command no longer flushes the Expires field for role credentials
    or the role History

### Changes

 * `flush` now flushes the STS IAM Role credentials first by default #236
 * Guided setup now uses the hostname or FQDN instead of full URL for the SSO StartURL #258

### New Features

 * Add a lot more `ProfileFormat` functions via sprig #244
 * `flush` command gives users more control over what is flushed
 * Add documentation for `SourceIdentity` for AssumeRole operations
 * Add `EnvVarTags` config file option #134

## [1.7.0] - 2022-01-09

### New Features
 * Add `Via` and `SSO` to possible `list` command output fields
 * Add `SSO` to list of valid ProfileFormat template variables
 * Improve ProfileFormat documentation
 * Add `config` command to manage `~/.aws/config` #157
 * Add Quick Start Guide
 * `console` command now works with any credentials using `$AWS_PROFILE` #234

### Bug Fixes

 * Fix broken FirstItem and StringsJoin ProfileFormat functions
 * Default ProfileFormat now zero-pads the AWS AccountID
 * Fix crash with invalid History tags

### Changes

 * `eval` command now supports `--url-action=print`

## [1.6.1] - 2021-12-31

### New Features
 * The `Via` role option is now a searchable tag #199
 * The `tags` command now returns the keys in sorted order

### Bug Fixes
 * Consistently pad AccountID with zeros whenever necessary
 * Detect role chain loops using `Via` #194
 * AccountAlias/AccountName tags are inconsistenly applied/missing #201
 * Honor config.yaml `DefaultSSO` #209
 * Setup now defaults to `warn` log level instead of `info` #214
 * `console` command did not know when you are using a non-Default SSO instance #208
 * cache now handles multiple AWS SSO Instances correctly which fixes numerous issues #219

### Changes
 * Reduce number of warnings #205

## [1.6.0] - 2021-12-24

### Breaking Changes
 * Fix issue with missing colon in parsed/generated Role ARNs for missing AWS region #192

### New Features
 * Setup now prompts for `LogLevel`
 * Suppress bogus warning when saving Role credentials in `wincred` store #183
 * Add support for role chaining using `Via` tag #38
 * Cache file is now versioned for better compatibility across versions of `aws-sso` #195

### Bug Fixes
 * Incorrect `--level` value now correctly tells user the correct name of the flag
 * `exec` command now uses `cmd.exe` when no command is specified

## [v1.5.1] - 2021-12-15

### New Features
 * Setup now prompts for `HistoryMinutes` and `HistoryLimit`

### Bug Fixes
 * Setup now uses a smaller cursor which doesn't hide the character
 * Fix setup bug where the SSO Instance was always called `Default`
 * Setup no longer accepts invalid characters for strings #178
 * Fix error/bell sound on macOS when selecting options during setup #179

## [v1.5.0] - 2021-12-14

### New Features
 * Add `HistoryMinutes` option to limit history by time, not just count #139

### Changes
 * Now use macOS `login` Keychain instead of `AWSSSOCli` #150
 * All secure storage methods now store a single entry instead of multiple entries
 * Replace `console --use-sts` with `console --prompt` #169
 * Improve password prompting for file based keyring #171

### Bug Fixes
 * file keyring will no longer infinitely prompt for new password

## [v1.4.0] - 2021-11-25

### Breaking Changes
 * Standardize on `AWS_SSO` prefix for environment variables
 * Remove `--region` flag for `eval` and `exec` commands
 * `console -use-env` is now `console --use-sts` to be more clear
 * Building aws-sso now requires Go v1.17+

### New Features
 * Add a simple wizard to configure aws-sso on first run if no ~/.aws-sso/config.yaml
	file exists
 * Update interactive selected item color schme to stand our better. #138
 * Add `eval --clear` and `eval --refresh`
 * Add full support for `DefaultRegion` in config.yaml
 * Add `--no-region` flag for `eval and `exec` commands
 * Add `process` command for AWS credential_process in ~/.aws/config #157
 * Add `ConsoleDuration` config option #159
 * Improve documentation of environment variables

### Bug Fixes
 * `exec` now updates the ENV vars of the forked processs rather than our own process
 * `eval` no longer prints URLs #145
 * Will no longer overwrite user defined AWS_DEFAULT_REGION #152
 * Fix bug where cache auto-refresh was not saving the new file, causing future
    runs to not utilize the cache
 * Remove `--duration` option from commands which don't support it
 * `LogLevel` and `UrlAction` in the config yaml now work #161
 * Add more unit tests & fix underlying bugs

## [v1.3.1] - 2021-11-15

 * Fix missing --url-action and  --browser #113
 * Don't print out URL when sending to browser/clipboard for security
 * Escape colon in ARN's for `-a` flag to work around the colon being a
    word delimiter for bash (auto)complete. #135
 * Add initial basic setup if there is a missing config.yaml #131

## [v1.3.0] - 2021-11-14

 * Add report card and make improvements to code style #124
 * Add auto-complete support #12
 * Add golangci-lint support & config file
 * Sort History tag based on time, not alphabetical
 * History entries now have how long since it was last used #123

## [v1.2.3] - 2021-11-13

 * Add support for tracking recently used roles via History tag for exec & console #29
 * Continue to improve unit tests
 * Fix bugs in `tags` command when using -A or -R to filter results
 * Fix missing tags when not defining roles in config.yaml #116
 * Fix bad Linux ARM64/AARCH64 rpm/deb packages with invalid binaries

## [v1.2.2] - 2021-11-11

 * Add `AccountAlias` and `Expires` to list of fields that can be displayed via
    the `list` command
 * `AccountAlias` replaces `AccountName` in the list of default fields for `list`
 * Add RPM and DEB package support for Linux on x86_64 and ARM64 #52

## [v1.2.1] - 2021-11-03

 * Add customizable color support #79
 * Simplify options for handling URLs and refactor internals #82
 * Rework how defaults are handled/settings loaded
 * Remove references to `duration` in config which don't do anything
 * Add additional config file options:
	- UrlAction
	- LogLevel
	- LogLines
	- DefaultSSO
 * Replace `--print-url` with `--url-action` #81
 * Add support for `DefaultRegion` in config file  #30
 * `console` command now supports `--region`
 * `list` command now reports expired and has constant sorting of roles #71
 * Fix bug where STS token creds were cached, but not reused.
 * `list -f` now sorts fields
 * Use cache for tracking when STS tokens expire
 * `exec` command now ignores arguments intended for the command being run #93
 * Remove `-R` as a short version of `--sts-refresh` to avoid collision with exec role #92
 * Fix finding $HOME directory on Windows and make GetHomePath() cross platform #100
 * Fix issue with AWS AccountID's with leading zeros.  #96
 * Optionally delete STS credentials from secure store cache #104
 * Add support for Brew #52

## [v1.2.0] - 2021-10-29

 * `console` command now can use ENV vars via --use-env #41
 * Fix bugs in `console` with invalid CLI parsing
 * Tag keys and values are now separate choices #49
 * Auto-complete options are now sorted
 * Started writing some unit tests
 * Do SSO authentication after role selection to improve performance
    even when we have cached creds
 * Add support for `AWS_SSO_PROFILE` env var and `ProfileFormat` in config #48
 * Auto-detect when local cache it out of date and refresh #59
 * Add support for `cache` command to force refresh AWS SSO data
 * Add support for `renew` command to refresh AWS credentials in a shell #63
 * Rename `--refresh` flag to be `--sts-refresh`
 * Remove `--force-refresh` flag from `list` command
 * Add role metadata when selecting roles #66

## [v1.1.0] - 2021-08-22

 * Move role cache data from SecureStore into json CacheStore #26
 * `exec` command will abort if a conflicting AWS Env var is set #27
 * Add `time` command to report how much time before the current STS token expires #28
 * Add support for printing Arn in `list` #33
 * Add `console` support to login to AWS Console with specified role #36
 * `-c` no longer is short flag for `--config`

## [v1.0.1] - 2021-07-18

 * Add macOS/M1 support
 * Improve documentation
 * Fix `version` output
 * Change `exec` prompt to work around go-prompt bug
 * Typing `exit` now exits without an error
 * Add help on how to exit via `exit` or ctrl-d

## [v1.0.0] - 2021-07-15

Initial release

[Unreleased]: https://github.com/synfinatic/aws-sso-cli/compare/v1.6.1...main
[v1.6.1]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.6.1
[v1.6.0]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.6.0
[v1.5.1]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.5.1
[v1.5.0]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.5.0
[v1.4.0]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.4.0
[v1.3.1]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.3.1
[v1.3.0]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.3.0
[v1.2.3]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.2.3
[v1.2.2]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.2.2
[v1.2.1]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.2.1
[v1.2.0]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.2.0
[v1.1.0]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.1.0
[v1.0.1]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.0.1
[v1.0.0]: https://github.com/synfinatic/aws-sso-cli/releases/tag/v1.0.0
