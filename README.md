# Mika

[![build status](http://ci.totdev.in/projects/2/status.png?ref=master)](http://ci.totdev.in/projects/2?ref=master)

A torrent tracker written in the Go programming language and using redis
as a backend database.


## Features

Makes cake

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
    hacker@nsa:$ mkdir -p $GOPATH/src/git.totdev.in/totv
    hacker@nsa:$ cd $GOPATH/src/git.totdev.in/totv
    hacker@nsa:$ git clone git@git.totdev.in:totv/mika
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