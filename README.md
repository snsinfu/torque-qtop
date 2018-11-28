# qtop

[![Release][release-badge]][release-url]
[![Build Status][travis-badge]][travis-url]
[![MIT License][license-badge]][license-url]

CUI job monitoring program for the torque resource manager.

[release-badge]: https://img.shields.io/github/release/snsinfu/torque-qtop.svg
[release-url]: https://github.com/snsinfu/torque-qtop/releases/latest
[travis-badge]: https://travis-ci.org/snsinfu/torque-qtop.svg?branch=master
[travis-url]: https://travis-ci.org/snsinfu/torque-qtop
[license-badge]: https://img.shields.io/badge/license-MIT-blue.svg
[license-url]: https://github.com/snsinfu/torque-qtop/blob/master/LICENSE.txt

## Install

Download the static 64-bit linux build `qtop` from [release page][release-url].
Put it into your `bin` directory and now you can use `qtop` command.

## Requirements

qtop itself is a statically linked pure go program and requires nothing. Works
with TORQUE 6.1.2 servers. `trqauthd` must be listening on unix domain socket
`/tmp/trqauthd-unix`.

## Build

```console
git clone https://github.com/snsinfu/torque-qtop
cd torque-qtop
go build -o ~/bin/qtop ./qtop/cmd
```

## Test

```console
go vet ./...
go test ./...
```

## Torque support

qtop is developed only for torque version 6.1.2. I won't support any future
versions because [torque went proprietary][torque-download].

[torque-download]: https://www.adaptivecomputing.com/support/download-center/torque-download/
