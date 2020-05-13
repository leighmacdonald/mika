# Implementing Mika Into Your Codebase

This document will describe the required actions you need to perform to have integration of the 
tracker into your own code bases. This document assumes that you are at least proficient enough
in your language of choice to implement the ideas expressed here. You do not need to be 
proficient in Go, but a good understanding of unix command line environment will make it easier.
We are also assuming you are using the tracker for a private site which has its own users and
its own database and authentication systems in place.


## Assumptions of What You Have

The following are a general guideline of what is needed to get running:

The tracker software itself, installable in a few different ways:
- Using binary releases (Not available currently)
- Building from source. Go 1.4+ environment, you can probably install from your distros package manager. If not you
can download the latest from the [go site](https://golang.org/dl/). The setup of this is beyond the scope of 
this document, but you can find [installation instruction](https://golang.org/doc/install) on the go site itself.

- A front end site of some sort to, like gazelle, or hopefully a better a custom site.
- This frontend should be tied to a RDBMS of some sort, mysql/postgres, to store and load torrent and user data.


## Loading Torrents 

Currently, mika does not support automatic torrent registration on announce by default. This is something that doesn't 
make much sense on a private site usually. Because of this we must tell it about any torrent
we want to track. This is done over the api endpoint `POST /api/torrent`. The payload should consist of the 
following:

    POST /api/torrent
    {
        'info_hash': "e940a7a57294e4c98f62514b32611e38181b6cae",
        'torrent_id': 123,
        'name': "Torrent.Name-GROUP"
    }
     
- **info_hash** refers to the .torrent files hex infohash value. This must be unique.
- **torrent_id** refers to your primary key of the torrent you are tracking within your database. This must be unique.
- **name** is the simple title of the torrent. This is used purely for extra information within the system.

We track both the info_hash and torrent_id internally because it makes certain operations easier and less costly at
the expense of a bit more memory usage.

You MUST keep this up to date by tying in your upload forms to call this API endpoint somehow. You can either do it
instantly in the request, or queue it up as a task for your system to execute. Its important this
happens quite fast as the tracker will reject any announce for the torrent until that time.

## Loading Users

Similar to the torrents, we also must get notified of users in the system via API requests.

    POST /api/user
    {
        'user_id': 123,
        'passkey': "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        'can_leech': true,
        'name': "b.gates"
    }
    
- **user_id** Users primary key tracked in your own RDBMS system. This must be unique.
- **passkey** The users passkey used to authenticate announce requests. This must be unique.
- **can_leech** true if the user can leech false otherwise.
- **name** The users displayed username.


## Configure whitelist

Since we by default only allow certain clients they must be loaded first. We only check for the
matching prefix when validating a client. Unfortunately determined people can easily spoof this if 
they wanted, so the ability to actually restrict other clients is limited in this regard. Don't expect
to catch cheaters with this alone. For a list of common prefixes, please see the [bt spec page](https://wiki.theory.org/BitTorrentSpecification)
.

    POST /api/whitelist
    {
        'prefix': "-DE",
        'client': "Deluge"
    }
    
## Updating Leecher & Seeder Counts

Keeping this data up to date required you to fetch the data from the API and store it in
your own database as leecher/seeder counts. Without this information being stored in your
own system it becomes very difficult to sort based on those attributes. Because of
this, its recommended to fetch this periodically and update your own counts.

    GET /api/counts
    
This returns an array of objects matching the following structure:

    [
        {
            'leechers': 0,
            'info_hash': '9c2f8f7f4996b2853509247504681dbe98e5d0c1',
            'snatches': 0,
            'seeders': 0,
            'torrent_id': 1112
        }, ...
    ]
    
