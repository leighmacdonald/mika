# Configuration

Configuration is handled through a simple JSON formatted file. When starting mika,
it will look for a config in the current directory called config.json unless a config
parameter is passing, in which case it will use that config. Due to the config being 
JSON formatted you cannot add comments to the file as this is not supported in JSON itself. This
is the annotated description of the options in lieu.


    {
    "Debug": false,
    
    
Enable debug output. You shouldn't normally use this unless you are diagnosing an issue.

    "ListenHost": ":34000",
    
Host and port that the tracker should listen to. If host is empty it will listen
on all available interfaces. It's currently best to set it to a specific ipv4 interface
so that unsupported ipv6 connections do not come in.

    "RedisHost": "localhost:6379",
    
Host and port to your redis instance.

    "RedisPass": "",

Optional redis password, leave as empty string if none.

    "RedisMaxIdle": 500,
    
Maximum number of redis connections to leave idle at all times.

    "AnnInterval": 300,
    
Suggested announce interval sent to the clients, in seconds. This has a massive influence on performance so
you must take care not to set this to a value that your server cannot handle.

    "AnnIntervalMin": 10,
    
Minimum announce interval the clients are able to send at.

    "ReapInterval": 400,
    
How long, in seconds, after we last received an announce to declare a peer as gone or disconnected.
This happens when users do not close their tracker clients cleanly so we do not receive the
stopped announce event and do not purge the peer properly from the peer list, or just losing connection. 
This value must be greater than `AnnInterval` and should be probably around 20%-50% more than `AnnInterval`. The closer
you make the values, the more accurate your peer list will be, but if you go too far you can potentially drop
valid peers too.

    "HNRThreshold": 1209600,
    
Amount of time that must pass before an incomplete torrent is declared as a HnR. The user can stop and 
start the torrent all they want without issue until this time has elapsed.
 
    "SQLHost": "localhost",
    "SQLPort": 3306,
    "SQLDB": "dbname",
    "SQLUser": "",
    "SQLPass": "",
    
MySQL host details used by warmup scripts

    "SentryDSN": "http://x:x@sentry-host.com/1"
    }
  
Optional sentry api details for storing error events. If empty sentry output will be
disabled.