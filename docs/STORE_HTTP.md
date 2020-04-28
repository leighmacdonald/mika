# HTTP Store

This storage interface provides a standard HTTP interface for querying and updating peer & torrent
data.

## store.TorrentStore

This describes the API for dealing with torrents.

This is a reasonably performant option as torrents to not need to be updated nearly as often
as peer data.

### TorrentStore.GetTorrent

    GET /api/torrent/<info_hash>
    {
        
    }


## store.PeerStore

This describes the API for dealing with peers / swarms.

Unless your tracker is small, its not recommended using this as there is more overhead (cost) 
associated with the encoding/decoding of all the data being transported across HTTP as JSON.

### PeerStore.GetPeers

    GET /api/torrent/<info_hash/peers[?limit=30]
    {
        "peers" []{Peer..}   
    }
    