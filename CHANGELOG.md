# Changelog


## v0.0.21 - 2025-09-30
### Fixed
* Race in restartHandler during SIGTERM cancellation

## v0.0.20 - 2025-09-29
### Changed
* Total node progress is logged in between --delay-between-restarts too
### Fixed
* SIGINT \ SIGTERM now interrupts, not waiting for `delay-between-intervals` to complete

## v0.0.19 - 2025-03-17
### Fixed
* storage pod selection by node id in k8s
* update ssl certificate for tests

## v0.0.18 - 2025-03-14
### Added
* Add new target into Makefile - install
* New --cleanup-on-exit flag for `ydbops restart`. If specified, will intercept SIGTERM\SIGINT, safely finish current CMS batch and exit, cleaning up request
### Changed
* bumped Go version to 1.23
### Removed
* Removed an unimplemented --continue flag. Turned out it's not necessary, filtering by uptime etc is expressive and much simpler
### Fixed
* 'delay-between-restarts' applies between CMS batches as well
* Returned forgotten bracket in <subcommand> help message

## v0.0.17 - 2025-01-22
### Fixed
* --started option help contained a critically misleading typo (> vs <)

## v0.0.16 - 2024-12-20
### Changed
* Downgraded dependencies

## v0.0.15 - 2024-12-18
### Added
* nodes-inflight option, that limits inflight restarts
* delay-between-restarts option that adds wait between two consecutive restarts
### Changed
* Endpoint non required

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
