package mysql

import (
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
)

// PeerStore is the mysql backed implementation of store.PeerStore
type PeerStore struct {
	db *sqlx.DB
}

// Close will close the underlying database connection
func (ps *PeerStore) Close() error {
	return ps.db.Close()
}

// Update will sync the new peer data with the backing store
func (ps *PeerStore) Update(_ model.InfoHash, _ *model.Peer) error {
	panic("implement me")
}

// Add insets the peer into the swarm of the torrent provided
func (ps *PeerStore) Add(ih model.InfoHash, p *model.Peer) error {
	const q = `
	INSERT INTO peers 
	    (peer_id, info_hash, addr_ip, addr_port, location, user_id)
	VALUES 
	    (:peer_id, :info_hash, :addr_ip, :addr_port, :location, :user_id)
	`
	_, err := ps.db.Exec(q, p.PeerID, ih, p.IP, p.Port, p.Location, p.UserID)
	if err != nil {
		return err
	}
	return nil
}

// Delete will remove a peer from the swarm of the torrent provided
func (ps *PeerStore) Delete(ih model.InfoHash, p *model.Peer) error {
	const q = `DELETE FROM peers WHERE info_hash = ? AND peer_id = ?`
	_, err := ps.db.Exec(q, ih, p.PeerID)
	return err
}

// Get will fetch the peer from the swarm if it exists
func (ps *PeerStore) Get(ih model.InfoHash, peerID model.PeerID) (*model.Peer, error) {
	const q = `SELECT * FROM peers WHERE info_hash = ? AND peer_id = ? LIMIT 1`
	var peer model.Peer
	if err := ps.db.Get(&peer, q, ih, peerID); err != nil {
		return nil, errors.Wrap(err, "Unknown peer")
	}
	return &peer, nil
}

// GetN will fetch the torrents swarm member peers
func (ps *PeerStore) GetN(ih model.InfoHash, limit int) (model.Swarm, error) {
	const q = `SELECT * FROM peers WHERE info_hash = ? LIMIT ?`
	var peers []*model.Peer
	if err := ps.db.Select(&peers, q, ih, limit); err != nil {
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
