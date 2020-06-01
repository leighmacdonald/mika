package mysql

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

// PeerStore is the mysql backed implementation of store.PeerStore
type PeerStore struct {
	db *sqlx.DB
}

func (ps *PeerStore) Name() string {
	return driverName
}

// Sync batch updates the backing store with the new PeerStats provided
func (ps *PeerStore) Sync(b map[store.PeerHash]store.PeerStats, cache *store.PeerCache) error {
	const q = `CALL peer_update_stats(?, ?, ?, ?, ?, ?)`
	tx, err := ps.db.Begin()
	if err != nil {
		return errors.Wrap(err, "Failed to being user Sync() tx")
	}
	stmt, err2 := tx.Prepare(q)
	if err2 != nil {
		return errors.Wrap(err2, "Failed to prepare user Sync() tx")
	}
	for ph, stats := range b {
		ih := ph.InfoHash()
		pid := ph.PeerID()
		if _, err := stmt.Exec(ih.Bytes(), pid.Bytes(), stats.Downloaded, stats.Uploaded,
			stats.Announces, stats.LastAnnounce); err != nil {
			if err := tx.Rollback(); err != nil {
				log.Errorf("Failed to roll back peer Sync() tx")
			}
			return errors.Wrap(err, "Failed to exec peer Sync() tx")
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "Failed to commit user Sync() tx")
	}
	return nil
}

// Reap will loop through the peers removing any stale entries from active swarms
func (ps *PeerStore) Reap(cache *store.PeerCache) {
	const q = `CALL peer_reap(?)`
	rows, err := ps.db.Exec(q, time.Now().Add(-15*time.Minute))
	if err != nil {
		log.Errorf("Failed to reap peers: %s", err.Error())
		return
	}
	count, err2 := rows.RowsAffected()
	if err2 != nil {
		log.Errorf("Failed to get reap count: %s", err2)
	}
	log.Debugf("Reaped %d peers", count)
}

// Close will close the underlying database connection
func (ps *PeerStore) Close() error {
	return ps.db.Close()
}

// Add insets the peer into the swarm of the torrent provided
func (ps *PeerStore) Add(ih store.InfoHash, p store.Peer) error {
	const q = `CALL peer_add(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	point := fmt.Sprintf("POINT(%s)", p.Location.String())
	_, err := ps.db.Exec(q, ih.Bytes(), p.PeerID.Bytes(), p.UserID, p.IP.String(), p.Port, point,
		p.AnnounceFirst, p.AnnounceLast, p.Downloaded, p.Uploaded, p.Left, p.Client)
	if err != nil {
		return err
	}
	return nil
}

// Delete will remove a peer from the swarm of the torrent provided
func (ps *PeerStore) Delete(ih store.InfoHash, p store.PeerID) error {
	const q = `CALL peer_delete(?, ?)`
	_, err := ps.db.Exec(q, ih.Bytes(), p.Bytes())
	return err
}

// Get will fetch the peer from the swarm if it exists
func (ps *PeerStore) Get(peer *store.Peer, ih store.InfoHash, peerID store.PeerID) error {
	const q = `CALL peer_get(?, ?)`
	if err := ps.db.Get(peer, q, ih.Bytes(), peerID.Bytes()); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return consts.ErrInvalidPeerID
		}
		return errors.Wrap(err, "Error looking up peer")
	}
	return nil
}

// GetN will fetch the torrents swarm member peers
func (ps *PeerStore) GetN(ih store.InfoHash, limit int) (store.Swarm, error) {
	const q = `CALL peer_get_n(?, ?)`
	swarm := store.NewSwarm()
	rows, err := ps.db.Query(q, ih.Bytes(), limit)
	if err != nil {
		return swarm, err
	}
	var p store.Peer
	var ip string
	for rows.Next() {
		if err := rows.Scan(&p.PeerID, &p.InfoHash, &p.UserID, &ip, &p.Port, &p.Downloaded, &p.Uploaded,
			&p.Left, &p.TotalTime, &p.Announces, &p.SpeedUP, &p.SpeedDN, &p.SpeedUPMax, &p.SpeedDNMax,
			&p.Location, &p.AnnounceLast, &p.AnnounceFirst); err != nil {
			return swarm, err
		}
		p.IP = net.ParseIP(ip)
		swarm.Add(p)
	}
	return swarm, nil
}

type peerDriver struct{}

// New returns a mysql backed store.PeerStore driver
func (pd peerDriver) New(cfg interface{}) (store.PeerStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db := sqlx.MustConnect(driverName, c.DSN())
	return &PeerStore{
		db: db,
	}, nil
}

func init() {
	store.AddPeerDriver(driverName, peerDriver{})
}
