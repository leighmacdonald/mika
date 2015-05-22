# Cheater Detection

The following document will outline the currently known methods for cheating, as well
as the ways we will attempt to address detecting them.

## Detection methods

### Improbably Stats
 
A very simple detection that will trigger if anyone goes over a maximum speed threshold.
A good example would be to set this to a bit over 10Gb/s since its a safe assumption
that nobody is capable of seeding at those speeds as of 2015.

- Most users are not dumb enough to cheat with speeds this high and will just
 make their speeds a much more reasonable speed to evade detection.

### Uploading with no peers

Simple detection that simply watches a peer for transfer stats when no other peers have
reported data for the swarm within the announce interval*2 plus a buffer period

- Can't be detected easily in swarms of more than a few peers

### Empty Peer Sets

This method of detection functions by generating a fake list of peers and sending them
to a client as a peer list for the requested info hash. We then monitor this peers data
reporting stats and if they are changing still, we can determine that they are faking
their upload stats.

- Some better mods will not fake peer stats if the swarm speed is zero. We also need to
 fake this speed using fake peers if we are going to detect those.
 
### Historical Analysis

The peer stats will be recorded over time so that we can determine average speeds for 
a user at a specific IP. If we then detect a speed that is outside a tolerance the
user will get disabled. While this is not necessarily proof of a user cheating, it
does force the user to tell us that they are using a seedbox so that we can determine
if further action should take place.
 
- Can only detect changes over time, so the user has to have already been using the site at
their normal speeds.
- Needs a minimum number of downloads to be tracked before a reliable enough conclusion 
can be reached.
 
## Info sources

- http://www.seba14.org/
- http://www.sb-innovation.de/forum.php
- https://hellwich.nl/shu/main/