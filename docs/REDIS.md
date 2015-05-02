Redis Usage
===========

This document describes the storage layout used for the redis database.

Requirements
------------

For the peer lists to remain up to date we need to add the following settings
to redis.conf

notify-keyspace-events Kx

This settings tells redis to publish a key expire event over a channel which
we can intercept and then handle anything that needs to occur.

Standard Structures
-------------------

All tracker related entries in the redis db will use the t: prefix to signify
tracker.

In general a "t" or "p" in the key name will refer to a "torrent" or "peer".

**Info Hash Mappings**

The var $torrent_id refers to the torrents.id column in the SQL DB. These
hashes should be loaded in via the manage.py tool. The existence of one
of these keys determines the validity of an info hash received from an
announce. This means that the key must be flushed or added if the
torrent is created or deleted. The tracker itself does *not* manage this
itself.

[KEY] "t:info_hash:$info_hash" -> $torrent_id


**Whitelist**

The white list is a simple set of client prefixes and long

[HASH] "t:whitelist:$prefix" -> $long_client_name

**Users**

Users are mostly referred to by their unique passkey and not their user_id as we
do not care about any user information, just they they are allowed to participate
in our swarms

[HASH] "t:user:$passkey"

**Torrent Columns and Types**

- user_id int


**Torrent Key**

[HASH] "t:t:<info_hash>"

**Torrent Columns and Types**
- announces int
- leechers int
- seeders int
- snatches int

Torrent Peer Data in Hash Key

**Torrent Peer Key**

[HASH] t:p:<torrent_id>:<peer_id>

**Torrent Peer Columns and Types**

- speed_up f
- speed_dn f
- completed f
- user_id int
- first_announce int
- last_announce int
- total_time seconds
- active bool

**Torrent Peer Timeout**

There is a special key set that using an expiration date based on the
configured announce interval and a small buffer. Another process will
watch a special redis channel for key expiry events matching the key
format. To prevent this from happening, this key is refreshed on every
announce from the peer. If this key expires we process the peer_id, removing
it from the torrents active peer set and marking the peer as active = 0. This key
is removed upon a stopped event as well.

[SETEX] "t:ptimeout:$torrent_id:$peer_id" -> "$ReapInterval" "1"

**Torrent Peer Set**

There will potentially be a set used to store peers for a torrent, but this
may not be needed as scanning the keys could be fast enough.

[SET] t:tpeers:<info_hash>

**Peers Torrent Set**

A set containing the torrent_id of every torrent the peer is currently
seeding or leeching from.

[SET] "t:p:$peer_id:torrents" [torrent_id, ...]

**User Sets**

An active collection of the users torrent history

A set of torrents the user is actively participating in.

[SET] "t:u:$user_id:active

When a user completed a torrent its torrent_id gets added to here and removed from incomplete.

[SET] "t:u:complete"

All incomplete torrent_ids are in this set. This include HnRs torrent_ids.

[SET] "t:u:incomplete"

If a torrents total active time is less than the hnr threshold and the user is inactive
the torrent_id will be placed in here. It will not be removed from the incomplete set.

[SET] "t:u:hnr"

**Global Stats/Info**

These stats are very cheap to use since they are just static values so we can
use these in any number of place we want without worry for performance.

Global leechers count

t:stats:leechers

Global seeders count

t:stats:seeders

Global announce count

t:stats:announces

Global scrape count

t:stats:scrapes

Global requests count

t:stats:requests

Indexes
-------

We will be maintaining some indexed fields to give the ability to sort torrent by.

    [ZADD] t:i:leechers
    [ZADD] t:i:seeders
    [ZADD] t:i:snatches



Pub/Sub Channels
----------------

The tracker will expect the following publish events to happen externally (such as the
www side or task queue).


**Torrent Peers**

This channel is used by torrent peer swarms, if the key expires before the next
client announce we assume the client has gone away so we remove from the
active peer SET and lower the seeder or leecher counts.