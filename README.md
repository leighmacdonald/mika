# Mika

A torrent tracker written in the Go programming language and using redis
as a backend database.

This tracker is notably different from others like chihaya in that it caters specifically
to private tracker needs.


## Features

- Complete JSON/REST Api for interacting with the tracker on a separate authenticated
port differing from the standard tracker port. This port is configured for TLS1.2+ only.
- Redis backed storage engine with **no** support for RDBMS systems (mysql/postgres)
- Designed for private tracker needs, [chihaya](https://github.com/chihaya/chihaya) is a more suitable 
choice for public trackers.
- (Partial) IPv4 and IPv6 support with the ability to enable or disable the stacks.
- Smart peer selection based on geographic location.
- No database reads on incoming announces/scrapes.
- User bonus point system built into the tracker which is updated on each request instead of large batches.
- (Partial) Support for metrics and instrumentation backends (graphite/influxdb)
- Python based API Client maintained at [totv-python](https://github.com/ToTV/totv-python).
- (Soon) PHP based API Client
- Client whitelists for only allowing specific torrent clients
- Tested on Linux, should run anywhere the golang platform does.
- User authentication via passkey

## Usage

Create and configure a new `config.json` file from the `config.json.dist` example. 

Currently we are storing torrent data in MySQL. Because of this we need
to load the existing data into redis. This includes the following entities:

- torrent.info_hash We track torrents using torrent_id internally
- user.passkey To validate users and track their relationship to specific peer_id's
- whitelist.peer_id whitelist.vstring Load the current client white list

To do this we use the manage.py script with the warmup command as follows and run 
the tracker.

    hacker@nsa:$ ./manage.py warmup
    > Warming up redis data...
    > Loading whitelist...   12
    > Loading passkeys...    146
    > Loading torrents...    3819
    hacker@nsa:$ ./mika
    
We also need to generate SSL keys for the API listener as follows:
    
    hacker@nsa:$ ./manage.py genkey
    > Generating new keys...
    Generating a 1024 bit RSA private key
    ...
    
    
### Current Issues & Considerations
    
The tracker does not currently gracefully reconnect everything when it looses connection
 to its redis backend. If you ever restart redis, make sure to restart mika as well until
 this issue is resolved.    
### Compiling

If building a binary from source you will need a Go 1.4+ SDK installed.

To build, set your $GOPATH env var to $git_project_clone, install deps and go build.
    
    hacker@nsa:$ export GOPATH=$dir_path
    hacker@nsa:$ mkdir -p $GOPATH/src/github.com/leighmacdonald
    hacker@nsa:$ cd $GOPATH/src/github.com/leighmacdonald
    hacker@nsa:$ git clone git@github.com:leighmacdonald/mika
    hacker@nsa:$ cd mika 
    hacker@nsa:$ cp .hooks/pre-commit .git/hooks/
    hacker@nsa:$ ./ci_build.sh
    hacker@nsa:$ ./mika

### Run-time options:

* `-config <config.json>` - Path to config file. Default is ./config.json
* `-procs` - Number of processor cores to use. The default is $numcores-1, but you may want
to lower this if there is other contentious services running too.


### Signals

* `SIGUSR2` - Reload config


### Auto start

There are many ways to start the application as a service. Currently I am using
supervisord and have included an example configuration under the `docs` folder.