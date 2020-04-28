// Package mysql provides mysql/mariadb backed persistent storage
//
// NOTE this requires MySQL 8.0+ / MariaDB 10.5+ (maybe 10.4?) due to the POINT column type
// +build mysql

package mysql

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"mika/consts"
	"mika/model"
	"mika/store"
)

const (
	driverName = "mysql"
)

type TorrentStore struct {
	db *sqlx.DB
}

func (s *TorrentStore) Close() error {
	return s.db.Close()
}

func (s *TorrentStore) GetTorrent(hash model.InfoHash) (*model.Torrent, error) {
	const q = `SELECT * FROM torrent WHERE info_hash = ? AND is_deleted = false`
	var t *model.Torrent
	if err := s.db.Get(t, q); err != nil {
		return nil, err
	}
	return t, nil
}

type PeerStore struct {
	db *sqlx.DB
}

func (ps *PeerStore) Close() error {
	return ps.db.Close()
}

func (ps *PeerStore) UpdatePeer(tid *model.Torrent, p *model.Peer) error {
	panic("implement me")
}

func (ps *PeerStore) AddPeer(t *model.Torrent, p *model.Peer) error {
	const q = `
	INSERT INTO peers 
	    (peer_id, torrent_id, addr_ip, addr_port, location, user_id, created_on, updated_on)
	VALUES 
	    (:peer_id, :torrent_id, :addr_ip, :addr_port, :location, :user_id, now(), :updated_on)
	`
	res, err := ps.db.Exec(q, p.PeerId, t.TorrentID, p.IP, p.Port, p.Location, p.UserId)
	if err != nil {
		return err
	}
	lastId, err := res.LastInsertId()
	if err != nil {
		return errors.New("Failed to fetch insert ID")
	}
	p.UserPeerId = uint32(lastId)
	return nil
}

func (ps *PeerStore) DeletePeer(tid *model.Torrent, p *model.Peer) error {
	const q = `DELETE FROM peers WHERE user_peer_id = :user_peer_id`
	_, err := ps.db.NamedExec(q, p)
	return err
}

func (ps *PeerStore) GetPeers(t *model.Torrent, limit int) ([]*model.Peer, error) {
	const q = `SELECT * FROM peers WHERE torrent_id = ? LIMIT ?`
	var peers []*model.Peer
	if err := ps.db.Select(&peers, q, t.TorrentID, limit); err != nil {
		return nil, err
	}
	return peers, nil
}

func (ps *PeerStore) GetScrape(t *model.Torrent) {
	panic("implement me")
}

func (s *TorrentStore) AddTorrent(t *model.Torrent) error {
	if t.TorrentID > 0 {
		return errors.New("Torrent ID already attached")
	}
	const q = `INSERT INTO torrent (info_hash, release_name, created_on, updated_on) VALUES( ?, ?, ?, ?)`
	res, err := s.db.NamedExec(q, t)
	if err != nil {
		return err
	}
	lastId, err := res.LastInsertId()
	if err != nil {
		return errors.New("Failed to fetch insert ID")
	}
	t.TorrentID = uint32(lastId)
	return nil
}

func (s *TorrentStore) DeleteTorrent(t *model.Torrent, dropRow bool) error {
	if dropRow {
		const dropQ = `DELETE FROM torrent WHERE torrent_id = :torrent_id`
		_, err := s.db.NamedExec(dropQ, t)
		if err != nil {
			return err
		}
	} else {
		const updateQ = `UPDATE torrent SET is_deleted = 1 WHERE torrent_id = :torrent_id`
		_, err := s.db.NamedExec(updateQ, t)
		if err != nil {
			return err
		}
	}
	return nil
}

type torrentDriver struct{}

func (td torrentDriver) NewTorrentStore(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*store.Config)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	var db *sqlx.DB
	if c.Conn != nil {
		db = c.Conn
	} else {
		db = sqlx.MustConnect("mysql", c.DSN())
	}
	return &TorrentStore{
		db: db,
	}, nil
}

type peerDriver struct{}

func (pd peerDriver) NewPeerStore(cfg interface{}) (store.PeerStore, error) {
	c, ok := cfg.(*store.Config)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	var db *sqlx.DB
	if c.Conn != nil {
		db = c.Conn
	} else {
		db = sqlx.MustConnect("mysql", c.DSN())
	}
	return &PeerStore{
		db: db,
	}, nil
}

func init() {
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
}
