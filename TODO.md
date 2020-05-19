# TODO

## Next Release (v.2.0.0)

- Fully working announces
    - Verify peers and their states
    - Verify batch recording stats
    - Add many more unit/integration tests for this. Currently mostly just a skeleton.
- Fully working scrapes
    - Verifies results
    - More unit/integration tests
- Apply ratio multipliers
- Implementations for all currently support store interfaces with tests
    - http is mostly all that remains
    - Add test for all fields that are transmitted
    
    
## Future
- Implement cheater detection mechanisms
- 
- [BEP0015 UDP Tracker](http://bittorrent.org/beps/bep_0015.html)
- [BEP0007 IPv6 Peers](http://bittorrent.org/beps/bep_0007.html)
- Limit concurrent downloads for a user. This means having user classes/roles of some sort that can
have limits attached to them.
- Separate build env for docker img

## Maybe
- Clustering support
- [BEP0024 Tracker Returns External IP](http://bittorrent.org/beps/bep_0024.html)
- Enforce announce intervals. Dont send peers for people announcing too fast.
- Connectivity check. Test the connectivity (NAT-Traversal) for a user the first time their IP:Port is
announced. If failed, dont send peers.