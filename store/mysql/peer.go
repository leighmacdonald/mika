package mysql

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// PeerStore is the mysql backed implementation of store.PeerStore
type PeerStore struct {
	db *sqlx.DB
}

func (ps *PeerStore) Sync(b map[model.PeerHash]model.PeerStats) error {
	const q = `
		UPDATE 
			peers
		SET
			total_announces = (total_announces + ?),
		    total_downloaded = (total_downloaded + ?),
		    total_uploaded = (total_uploaded + ?),
		    announce_last = ?
		WHERE
			info_hash = ? AND peer_id = ?
	`
	tx, err := ps.db.Begin()
	if err != nil {
		return errors.Wrap(err, "Failed to being user Sync() tx")
	}
	stmt, err := tx.Prepare(q)
	if err != nil {
		return errors.Wrap(err, "Failed to prepare user Sync() tx")
	}
	for ph, stats := range b {
		ih := ph.InfoHash()
		pid := ph.PeerID()
		_, err := stmt.Exec(
			stats.Announces,
			stats.Downloaded,
			stats.Uploaded,
			stats.LastAnnounce,
			ih.Bytes(),
			pid.Bytes())
		if err != nil {
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

func (ps *PeerStore) Reap() {
	const q = `DELETE FROM peers WHERE announce_last <= (NOW() - INTERVAL 15 MINUTE)`
	rows, err := ps.db.Exec(q)
	if err != nil {
		log.Errorf("Failed to reap peers: %s", err.Error())
		return
	}
	count, err := rows.RowsAffected()
	if err != nil {
		log.Errorf("Failed to get reap count: %s", err.Error())
	}
	log.Debugf("Reaped %d peers", count)
}

// Close will close the underlying database connection
func (ps *PeerStore) Close() error {
	return ps.db.Close()
}

// Add insets the peer into the swarm of the torrent provided
func (ps *PeerStore) Add(ih model.InfoHash, p model.Peer) error {
	const q = `
	INSERT INTO peers 
	    (peer_id, info_hash, addr_ip, addr_port, location, user_id, announce_first, announce_last)
	VALUES 
	    (?, ?, INET_ATON(?), ?, ST_PointFromText(?), ?, ?, ?)
	`
	point := fmt.Sprintf("POINT(%s)", p.Location.String())
	_, err := ps.db.Exec(q, p.PeerID.Bytes(), ih.Bytes(), p.IP.String(), p.Port, point, p.UserID,
		p.AnnounceFirst, p.AnnounceLast)
	if err != nil {
		return err
	}
	return nil
}

// Delete will remove a peer from the swarm of the torrent provided
func (ps *PeerStore) Delete(ih model.InfoHash, p model.PeerID) error {
	const q = `DELETE FROM peers WHERE info_hash = ? AND peer_id = ?`
	_, err := ps.db.Exec(q, ih, p)
	return err
}

// Get will fetch the peer from the swarm if it exists
func (ps *PeerStore) Get(peer *model.Peer, ih model.InfoHash, peerID model.PeerID) error {
	const q = `
		SELECT 
		    peer_id, info_hash, user_id, INET_NTOA(addr_ip) as addr_ip, addr_port, 
		    total_downloaded, total_uploaded, total_left, total_time, total_announces, 
		    speed_up, speed_dn, speed_up_max, speed_dn_max, ST_AsText(location) as location, 
		    announce_last, announce_first 
		FROM peers WHERE info_hash = ? AND peer_id = ? LIMIT 1`
	if err := ps.db.Get(peer, q, ih.Bytes(), peerID.Bytes()); err != nil {
		log.Errorf("Failed to query peer: %s", err.Error())
		return errors.Wrap(err, "Unknown peer")
	}
	return nil
}

// GetN will fetch the torrents swarm member peers
func (ps *PeerStore) GetN(ih model.InfoHash, limit int) (model.Swarm, error) {
	const q = `
		SELECT 
			peer_id, info_hash, user_id, INET_NTOA(addr_ip) as addr_ip, addr_port, 
		    total_downloaded, total_uploaded, total_left, total_time, total_announces, 
		    speed_up, speed_dn, speed_up_max, speed_dn_max,  ST_AsText(location) as location, 
		    announce_last, announce_first 
		FROM 
		    peers 
		WHERE 
		    info_hash = ? 
		LIMIT
		    ?`
	var peers model.Swarm
	if err := ps.db.Select(&peers, q, ih.Bytes(), limit); err != nil {
		return nil, err
	}
	return peers, nil
}

type peerDriver struct{}

// NewPeerStore returns a mysql backed store.PeerStore driver
func (pd peerDriver) NewPeerStore(cfg interface{}) (store.PeerStore, error) {
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
