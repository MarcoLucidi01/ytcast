changelog
=========

## v1.1.0

2022-02-04

- added `-a` (add) flag which allows to add videos to the queue without changing
  what's currently playing (`1270918`, `b08e793`). the implementation is not
  very pretty due to the Lounge api, but gets the job done (most of the times).
- `ytcast-bin` is available on the Arch User Repository (AUR)! this will make it
  easier for Arch Linux users to install the program. I'd like to thank the
  maintainer and julianYaman for letting me know. a link to the AUR package has
  been added in the readme (`b7741d7`).

## v1.0.0

2022-01-29

- `-l` flag is used now to list cached devices without getting an error (`4f19d6c`),
  while `-p` is for selecting the last used device (`02f8ec8`).
  this is an incompatible change so here we are at `v1.0.0`, the first major release!
- renamed `-n` flag to more intuitive `-d` (device) (`9368df8`). `-n` can still
  be used but is deprecated.
- added a quick install script for binaries in the readme (`b3726d6`, `96af820`)
  and various other readme updates e.g. a TOC (`ca36350`).

## v0.5.0

2022-01-19

- added a `release` script to automatically create new tag versions and GitHub
  releases (this release is actually the first one made with this script so it's
  kind of a final test for it).
- pre-compiled binaries for different architectures are built with the new
  `makefile` target `cross-build` (`34fcee5`) and are attached to the GitHub
  release (`7d3910a`).
- all the binaries built with the `makefile` are now statically linked and
  stripped (`626dc57`).

## v0.4.0

2022-01-10

- various DIAL and SSDP implementation improvements (`1a4671e`, `c15daaa`, `5fdbcb1`).
- print also initial part of USN (unique service name) when showing devices (`e24deb0`).
- if `-n` doesn't match anything trigger a re-discover (`e4932b0`).
- `-s` can now be used along with `-n` (`e4932b0`).
- added `-c` (clear cache) flag (`d60cb9f`).

## v0.3.0

2022-01-08

- removed microseconds from `-verbose` log (`5b96d81`).
- rediscover device after Wake-On-Lan since ip address and ports can change (`bff10f5`).
- use `http.Client` with proper timeout (`e390f0b`).

## v0.2.0

2022-01-05

- exit with error if more than one device matches `-n` (`3f7f820`).
- `readme.md` is no longer a draft!
- added `install` and `uninstall` targets to `makefile` (`e9c96d3`).
- use `$USER@$HOSTNAME` as "connect" name (`a518a67`).
- fixed some YouTube Lounge api calls that stopped working (`f3fde46`).

## v0.1.0

2021-12-14

- repo is public! this is the initial version, core functionality works!
