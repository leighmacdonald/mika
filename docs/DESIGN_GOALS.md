# Design Goals

The primary goal of mika is to have a modern, customizable, safe and high performance bittorrent tracker. It is
mainly designed for use as a private tracker, however it has the capability to operate as a public tracker
if desired. 

The reason for creating this is existing solutions are generally not very flexible. For example, they may require 
a specific SQL schema which is difficult to change or adapt to other front-ends (gazelle, unit3ed, xbt, etc.). Most
are also written in unsafe languages (C && C++) making them *potentially* vulnerable to memory related security issues.
We can avoid this entire class of problems by using a garbage collected language like golang.
 
An API exists to facilitate live administration and optional 2-way data communication between a web frontend 
like gazelle or unit3d for example. The cli interface `./mika client -h` communicates over this API.

## Memory Usage

Since mika is targeting private tracker use, we are selecting types appropriate for optimizing memory usage (size). Wherever
it makes sense we will use the smallest int types (uint16, uint32 mostly) needed for the value. This has the downside of
*potentially* slightly less optimized execution by the CPU. Private trackers are often strained
for resources already. We think it makes sense to take this approach since our intent is to store as much data in memory as 
possible and memory is often more expensive than CPU resources. This approach is most beneficial for tracking peer swarms 
which could be millions of entries in total across all the torrents.

## Storage Backend

Due to the way trackers work, there is a significant amount of database load created without a sound
model for caching. Many (most) existing trackers are making SELECT queries on every announce request. This
has severe drawbacks for trackers with little financial support since the huge load means having to spend
more on hardware or increasing announce intervals so much, the user experience suffers.

The storage can be defined as two discreet datasets, `Peers` and `Torrents` which are described in more detail below.  

### Torrents Store
 
SHOULD be backed by persistent storage, so it can survive restarts. Postgres/MariaDB(MySQL) are the recommended
choices for this data store.
 
Redis with AOF persistence is an acceptable choice if you are not running a RDBMS but still want durable storage. The 
redis RDB persistence option is subject to data loss on power failure so its not recommend to use if you care 
about your data consistency.

Custom storage adapters can be created simply by implementing the `store.TorrentStore` interface.

An HTTP storage backend also will be available to simplify integration for users who are not familiar with or
do not want to write a custom storage adapter in Go.

### Peers Store

MAY be backed by persistent storage, but high performance backend are ***highly recommended***. You should
not be storing ephemeral peer data in a RDBMS unless you have relatively low user counts. There is very few reasons to
actually persist this data beyond something like [redis RDB](https://redis.io/topics/persistence) which can
be useful to not have to "warm-up" data after a restart.

Custom storage adapters can be created simply by implementing the `store.PeerStore` interface.

#### HTTP Store

This is a special and unique option in the tracker space. Instead of storing data in a database directly, An external 
API conforming to our specification can be called when the tracker wants to read or write any data. This operates similar
to a [WebHook](https://en.wikipedia.org/wiki/Webhook) or callback. The http service can store the data in any 
way it desires as long as the API conforms. A basic example of the data flow is shown below.
    
    # Get a requested torrent from the frontend API service
    Tracker <- GET https://frontend.com/api/torrent/<info_hash>
    # .. handle some announces and trigger a sync for the data after some time .. 
    Tracker -> POST http://frontend.com/api/torrent/<info_hash>/peers
    Tracker -> POST http://frontend.com/api/torrent/<info_hash>/peers
    Tracker -> POST http://frontend.com/api/torrent/<info_hash>/peers
    # Rollup some stats into the main torrent like total_completed 
    # The torrent meta data doesnt need to be updated as often as peer state
    Tracker -> PATCH http://frontend.com/api/torrent/<info_hash>
    
Currently, JSON is the only planned data exchange format for HTTP. In the future other formats 
( [CapNProto](https://capnproto.org/), [MsgPack](https://msgpack.org/index.html), 
[protobuf](https://github.com/protocolbuffers/protobuf)) could be implemented if the demand
is there.

## Cheater Detection

Since mika is targeting private tracker use, We intend to implement various cheater detection methods to ensure
everyone is playing fair. This is not a real concern for public trackers because there is no reason, or consequences of
cheating on those systems.

See [CHEATERS.md](CHEATERS.md) for more detailed descriptions of methods used.

## Smart Peer Selection Strategies

There are several methods used to enable smarter, and configurable, peer selection for larger swarms. If your torrents
mostly have a swarm size less than the max returned peer count then it may make sense for your use case to simply 
disable them as the functionality is largely not useful.

All these methods MUST be sure to include enough completed (seeder) peers to ensure enough availability. DO NOT enable
all of these options as they may conflict with each other giving poorer results and if none were used. 

Except for location bias, these peer selections should only really apply to peers who have completed the download 
and are just seeding. People still actively downloading shouldn't be restricted or penalized. 

Whichever you select, if any, be sure it matches your goals as a tracker operator. Do not just select what you think
would be nice to have without considerations. For example, Some will benefit trackers with mostly small torrents, 
like ebooks or MP3, which can often be difficult for users to maintain a ratio on.

### Location Bias Strategy

By using a geo database ([MaxMind City](https://dev.maxmind.com/geoip/)) to lookup IP locations we can select 
peers that are globally closest to each other. We are calculating distances between the lat/long of peers under the 
[WGS84 (EPSG:4326/geographic)](https://en.wikipedia.org/wiki/World_Geodetic_System) projection. 

This should be considered the "fairest" option for most setups and should always be safe to enable.

### Completion Bias Strategy

To prevent what people may consider a "Pay2Win" scenario where User_A has a 10Gb+ SSD backed client in a 
data center (seedbox) and User_B at home on his 1Mb upload speed. We can negatively
bias users on how much they have uploaded after completing a torrent. This is designed to help users with 
lesser connections have less trouble maintaining an acceptable ratio based on the sites rules. This would not really
be recommended for a ratioless tracker.

### Peer Speed Bias Strategy

Similar in function to the completion bias, it will be able to either positively or negatively bias users based
on their peer speed. By favouring slower peers you are largely biasing your peers to non-seedbox backed home 
connections. This is of course not always the case, 1Gb home connections are more and more common these days. The 
opposite will favour low latency fast peers found in data centers (seedboxes).

### User Bias Strategy

User bias enables specific user classes to be prioritized. For example, you have a group of users defined as 
"uploaders". These users all have known fast connections and are often the original seeder of a torrent. You can 
ensure they are always in the swarm either to promote max download speeds or to encourage more uploads by those users. 

Another option could be if You want favour donators who help support the tracker or new users to help them get 
established, especially if you are not ratioless.


### Seed Time Bias Strategy

This involves prioritizing seeders who have been in the swarm the least amount of time. This is designed to help users
who are a bit late snatching a new release still be able to build some ratio.
