// Package tracker implements the public facing HTTP interfaces to the tracker.
//
// Tracker endpoints:
//
//	 - /:passkey/announce
//   - /:passkey/scrape
//
// API routes:
//
//  - General
//    - POST /ping
//    - GET /tracker/stats
//    - PATCH /config
//
//	- Torrents
//    - DELETE /torrent/:info_hash
//    - PATCH /torrent/:info_hash
//    - POST /torrent
//    - POST /whitelist
//    - GET /whitelist
//    - DELETE/whitelist/:prefix
//
//	- Users
//    - POST /user
//    - DELETE /user/pk/:passkey
//
package tracker
