# Mika - Bittorrent Tracker

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fleighmacdonald%2Fmika.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fleighmacdonald%2Fmika?ref=badge_shield)

[![Codacy Badge](https://api.codacy.com/project/badge/Grade/f06234b0551a49cc8ac111d7b77827b2)](https://www.codacy.com/manual/leighmacdonald/mika?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=leighmacdonald/mika&amp;utm_campaign=Badge_Grade)
[![Maintainability](https://api.codeclimate.com/v1/badges/4e3242de961462b0edc7/maintainability)](https://codeclimate.com/github/leighmacdonald/mika/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/4e3242de961462b0edc7/test_coverage)](https://codeclimate.com/github/leighmacdonald/mika/test_coverage)
[![Go Report Card](https://goreportcard.com/badge/github.com/leighmacdonald/mika)](https://goreportcard.com/report/github.com/leighmacdonald/mika)
[![GoDoc](https://godoc.org/github.com/leighmacdonald/mika?status.svg)](https://pkg.go.dev/github.com/leighmacdonald/mika)
![Lines of Code](https://tokei.rs/b1/github/leighmacdonald/mika)
[![Discord chat](https://img.shields.io/badge/discord-Chat%20Now-a29bfe.svg?style=flat-square)](https://discord.gg/jWXFcHW)

Mika is a torrent tracker written in the Go programming language.

It is designed exclusively for private tracker needs, [chihaya](https://github.com/chihaya/chihaya) is a more suitable 
choice for public trackers.

For the previous, 1.x code, see the [legacy branch](https://github.com/leighmacdonald/mika/tree/legacy).

## Documentation

The current documentation is within the [docs](docs) folder. Keep in mind that these are currently either out
of date with the current build, or referencing things that are not yet fully implemented.

## Support & Discussion

There is currently a discord server setup for mika. You can join [here](https://discord.gg/jWXFcHW). 

## Features (Planned)

A high level view of the features we integrate into the tracker. Some are fully implemented already, some are still in the works.

- REST JSON API for interacting with the tracker on a separate authenticated
port differing from the standard tracker port. This port is configured for TLS1.2+ only.
- CLI for interacting with the running tracker `./mika client -h`
- Multiple storage backends which can be selected based on needs and system architecture. You can define completely different stores
    for the 3 types of backend interfaces we implement: Users, Torrents, Peers.
    - `postgres` A PostgreSQL 10+ backed store. We also use the [PostGIS](https://postgis.net/) extension to store location
     data and perform geo queries.
    - `mysql/mariadb` A MySQL 5.1+ / MariaDB 10.1+ backed persistent storage backend. We use the POINT column for geospatial
    queries which is why we require these versions at minimum.
    - `redis` Redis provides an in-memory datastore which does get persisted to disk (if enabled in redis).
    - `memory` A simple in-memory storage which is not persisted anywhere.
    - `http` By implementing compatible REST API endpoints on your own HTTP service, we can communicate over REST with your own independent system. This is a good
    option if you don't know Go or do not want to change  your database schema to be compatible. Several authentication methods will be implemented.
    - `custom` You can easily add support for your own storage backends by implementing store.UserStore, store.PeerStore or store.TorrentStore interfaces as needed. PRs for
     new implementations welcomed.

- IPv4 and IPv6 support with the ability to enable or disable the stacks. Note that v4 requests will only return v4 peers, same applies to v6.
- Optional smarter peer selection [strategies](docs/DESIGN_GOALS.md).
- Either a single datastore read (which is cached, no future reads for the same resource made) or no database reads, depending on storage backends chosen on incoming announces/scrapes.
- User bonus point system built into the tracker which is updated on each request instead of large batches.
- Go/PHP based API Client examples
- Client whitelists for only allowing specific torrent clients
- Multi platform support. Should run on anything that go can target.
- User authentication via passkey
- Docker images for deployment

Some things we don't currently have plans to support:

- Non-compact responses. There is no reason to use non-compact responses for a private tracker. All modern and usual 
whitelisted clients support it.
- DHT bootstrapping node
- Migrations from existing tracker systems


## Build Notes

To also build the demo http server add the demos build tags.

    go build -tags demos 
    ./mika demoapi

## Usage

None, Don't use this yet.


## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fleighmacdonald%2Fmika.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fleighmacdonald%2Fmika?ref=badge_large)