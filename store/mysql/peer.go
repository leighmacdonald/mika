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
	"strings"
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
func (ps *PeerStore) Sync(b map[store.PeerHash]store.PeerStats) error {
	const q = `CALL peer_update_stats(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	tx, err := ps.db.Begin()
	if err != nil {
		return errors.Wrap(err, "Failed to being user Sync() tx")
	}
	stmt, err2 := tx.Prepare(q)
	if err2 != nil {
		return errors.Wrap(err2, "Failed to prepare user Sync() tx")
	}
	for ph, stats := range b {
		sum := stats.Totals()
		if _, err := stmt.Exec(ph.InfoHash().Bytes(), ph.PeerID().Bytes(),
			sum.TotalDn, sum.TotalUp, len(stats.Hist), sum.LastAnn,
			sum.SpeedDn, sum.SpeedUp, sum.SpeedDnMax, sum.SpeedUpMax); err != nil {
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
// TODO fetch peer hashes for expired peers to flush local caches
func (ps *PeerStore) Reap() []store.PeerHash {
	var peerHashes []store.PeerHash
	const q = `CALL peer_reap(?)`
	rows, err := ps.db.Exec(q, time.Now().Add(-15*time.Minute))
	if err != nil {
		log.Errorf("Failed to reap peers: %s", err.Error())
		return nil
	}
	count, err2 := rows.RowsAffected()
	if err2 != nil {
		log.Errorf("Failed to get reap count: %s", err2)
	}
	log.Debugf("Reaped %d peers", count)
	return peerHashes
}

// Close will close the underlying database connection
func (ps *PeerStore) Close() error {
	return ps.db.Close()
}

// Add insets the peer into the swarm of the torrent provided
func (ps *PeerStore) Add(ih store.InfoHash, p store.Peer) error {
	const q = `CALL peer_add(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	point := fmt.Sprintf("POINT(%s)", p.Location.String())
	ip6 := strings.Count(p.IP.String(), ":") > 1
	_, err := ps.db.Exec(q, ih.Bytes(), p.PeerID.Bytes(), p.UserID, ip6, p.IP.String(), p.Port, point,
		p.AnnounceFirst, p.AnnounceLast, p.Downloaded, p.Uploaded, p.Left, p.Client,
		p.CountryCode, p.ASN, p.AS, int(p.CryptoLevel))
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
		if err := rows.Scan(&p.PeerID, &p.InfoHash, &p.UserID, &p.IPv6, &ip, &p.Port, &p.Downloaded, &p.Uploaded,
			&p.Left, &p.TotalTime, &p.Announces, &p.SpeedUP, &p.SpeedDN, &p.SpeedUPMax, &p.SpeedDNMax,
			&p.Location, &p.AnnounceLast, &p.AnnounceFirst, &p.CountryCode, &p.ASN, &p.AS, &p.CryptoLevel); err != nil {
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
	db, err := sqlx.Connect(driverName, c.DSN())
	if err != nil {
		return nil, err
	}
	return &PeerStore{
		db: db,
	}, nil
}

func init() {
	store.AddPeerDriver(driverName, peerDriver{})
}
