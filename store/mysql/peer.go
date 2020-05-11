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

func (ps *PeerStore) Reap() {
	log.Debugf("Implemented mysql peer reaper")
}

// Close will close the underlying database connection
func (ps *PeerStore) Close() error {
	return ps.db.Close()
}

// Update will sync the new peer data with the backing store
func (ps *PeerStore) Update(_ model.InfoHash, _ model.Peer) error {
	panic("implement me")
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
