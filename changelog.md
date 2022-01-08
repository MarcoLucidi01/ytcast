changelog
=========

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
