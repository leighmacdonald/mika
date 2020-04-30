# Mika

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) 
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/f06234b0551a49cc8ac111d7b77827b2)](https://www.codacy.com/manual/leighmacdonald/mika?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=leighmacdonald/mika&amp;utm_campaign=Badge_Grade)
[![Discord chat](https://img.shields.io/badge/discord-Chat%20Now-a29bfe.svg?style=flat-square)](https://discord.gg/jWXFcHW)

A torrent tracker written in the Go programming language.

It is designed for private tracker needs, [chihaya](https://github.com/chihaya/chihaya) is a more suitable 
choice for public trackers.

For the previous, 1.x code, see the [legacy branch](https://github.com/leighmacdonald/mika/tree/legacy).

## Support & Discussion

There is currently a discord server setup for mika. You can join [here](https://discord.gg/jWXFcHW). 

## Features (Planned)

- Complete JSON/REST Api for interacting with the tracker on a separate authenticated
port differing from the standard tracker port. This port is configured for TLS1.2+ only.
- Multiple storage backends
    - PostgreSQL
    - MySQL/MariaDB
    - Redis
    - Memory
- IPv4 and IPv6 support with the ability to enable or disable the stacks.
- Smart peer selection based on geographic location.
- No database reads on incoming announces/scrapes.
- User bonus point system built into the tracker which is updated on each request instead of large batches.
- PHP based API Client
- Client whitelists for only allowing specific torrent clients
- Multi platform support
- User authentication via passkey

## Build Notes

To also build the demo http server add the demos build tags.

    go build -tags demos 
    ./mika demoapi

## Usage

None, Don't use this yet.
