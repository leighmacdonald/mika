# Mika

A torrent tracker written in the Go programming language.

It is designed for private tracker needs, [chihaya](https://github.com/chihaya/chihaya) is a more suitable 
choice for public trackers.

For the previous, 1.x code, see the [legacy branch](https://github.com/leighmacdonald/mika/tree/legacy).

## Features

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

## Usage

None, Don't use this yet.
