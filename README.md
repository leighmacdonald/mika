# Mika

A torrent tracker written in the Go programming language and using redis
as a backend database.


## Features

Makes cake

## Usage

Create and configure a new `config.json` file from the `config.json.dist` example. 

Currently we are storing torrent data in MySQL. Because of this we need
to load the existing data into redis. This includes the following entities:

- torrent.info_hash -> torrent.id We track torrents using torrent_id internally
- user.passkey -> user.id To validate users and track their relationship to specific peer_id's
- whitelist.peer_id -> whitelist.vstring Load the current client white list

To do this we use the manage.py script with the warmup command as follows and run 
the tracker.

    hacker@nsa:$ ./manage.py warmup
    > Warming up redis data...
    > Loading whitelist...   12
    > Loading passkeys...    146
    > Loading torrents...    3819
    hacker@nsa:$ ./mika
    
### Compiling

If building a binary from source you will need a Go 1.4+ SDK installed.

To build, set your $GOPATH env var to $git_project_clone, install deps and go build.

    
    hacker@nsa:$ cd $git_project_clone
    hacker@nsa:$ export GOPATH=$git_project_clone
    hacker@nsa:$ go get github.com/chihaya/bencode github.com/garyburd/redigo/redis github.com/kisielk/raven-go/raven github.com/labstack/echo github.com/influxdb/influxdb/client
    hacker@nsa:$ make
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