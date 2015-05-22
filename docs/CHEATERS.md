# Cheater Detection

The following document will outline the currently known methods for cheating, as well
as the ways we will attempt to address detecting them.

## Detection methods

### Empty Peer Sets

This method of detection functions by generating a fake list of peers and sending them
to a client as a peer list for the requested info hash. We then monitor this peers data
reporting stats and if they are changing still, we can determine that they are faking
their upload stats.

Known detections:

- Some better mods will not fake peer stats if the swarm speed is zero. We also need to
 fake this speed if we are going to detect those.
 
## Info sources

- http://www.seba14.org/
- http://www.sb-innovation.de/forum.php
- https://hellwich.nl/shu/main/