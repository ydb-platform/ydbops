# Changelog


## v0.0.14 - 2024-12-09
### Changed
* Active profile now uses 'current-profile' key in yaml config, rather than 'active_profile', to comply with the docs
### Fixed
* 'ydbops maintenance' command could not accept nodeIds in '--hosts' option (e.g. --hosts=1,2)
* 'ydbops maintenance' subtree should now properly use filters such as 'started', 'version' etc.

## v0.0.13 - 2024-11-05
### Fixed
* ydbops now properly continues the restart loop even if listing nodes during maintenance check fails with "retry exceeded" error

## v0.0.12 - 2024-10-31
### Changed
* migrated to changie for keeping a changelog

## 0.0.11
+ `--started` and `--version` no longer silently include nodes, which did not have this info supplied by CMS at all (due to old YDB version). 
  Now `ydbops` explicitly refuses to add any nodes with empty start time or empty version and produces a warning.

## 0.0.10
+ `version` command
+ new release pipeline - modify CHANGELOG.md only, rest is automatic

## 0.0.9
+ Information about version in help output
+ Scripts for build release
