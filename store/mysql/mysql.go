// Package mysql provides mysql/mariadb backed persistent storage
//
// NOTE this requires MySQL 8.0+ / MariaDB 10.5+ (maybe 10.4?) due to the POINT column type
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	// imported for side-effects
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

const (
	driverName = "mysql"
)

// ErrNoResults is the string returned from the driver when no rows are returned
const ErrNoResults = "sql: no rows in result set"

// UserStore is the MySQL backed store.UserStore implementation
type UserStore struct {
	db *sqlx.DB
}

func (u *UserStore) Name() string {
	return driverName
}

// Sync batch updates the backing store with the new UserStats provided
func (u *UserStore) Sync(b map[string]store.UserStats) error {
	const q = `CALL user_update_stats(?, ?, ?, ?)`
	// TODO use ctx for timeout
	ctx := context.Background()
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to being user Sync() tx")
	}
	stmt, err := tx.Prepare(q)
	if err != nil {
		return errors.Wrap(err, "Failed to prepare user Sync() tx")
	}
	for passkey, stats := range b {
		_, err := stmt.Exec(passkey, stats.Announces, stats.Uploaded, stats.Downloaded)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Errorf("Failed to roll back user Sync() tx")
			}
			return errors.Wrap(err, "Failed to exec user Sync() tx")
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "Failed to commit user Sync() tx")
	}

	return nil
}

// Add will add a new user to the backing store
func (u *UserStore) Add(user store.User) error {
	const q = `CALL user_add(?, ?, ?, ?, ?, ?, ?)`
	_, err := u.db.Exec(q, user.UserID, user.Passkey, user.DownloadEnabled,
		user.IsDeleted, user.Downloaded, user.Uploaded, user.Announces)
	if err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (u *UserStore) GetByPasskey(user *store.User, passkey string) error {
	const q = `CALL user_by_passkey(?)`
	if err := u.db.Get(user, q, passkey); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidUser
		}
		return errors.Wrap(err, "Could not query user by passkey")
	}
	return nil
}

// GetByID returns a user matching the userId
func (u *UserStore) GetByID(user *store.User, userID uint32) error {
	const q = `CALL user_by_id(?)`
	if err := u.db.Get(user, q, userID); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidUser
		}
		return errors.Wrap(err, "Could not query user by user_id")
	}
	return nil
}

// Delete removes a user from the backing store
func (u *UserStore) Delete(user store.User) error {
	if user.UserID == 0 {
		return errors.New("User doesnt have a user_id")
	}
	const q = `CALL user_delete(?)`
	if _, err := u.db.Exec(q, user.UserID); err != nil {
		return errors.Wrap(err, "Failed to delete user")
	}
	user.UserID = 0
	return nil
}

func (u *UserStore) Update(user store.User, oldPasskey string) error {
	const q = `CALL user_update(?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := u.db.Exec(q, user.UserID, user.Passkey, user.DownloadEnabled,
		user.IsDeleted, user.Downloaded, user.Uploaded, user.Announces,
		oldPasskey); err != nil {
		return errors.Wrapf(err, "Failed to update user")
	}
	return nil
}

// Close will close the underlying database connection and clear the local caches
func (u *UserStore) Close() error {
	return u.db.Close()
}

// TorrentStore implements the store.TorrentStore interface for mysql
type TorrentStore struct {
	db *sqlx.DB
}

func (s *TorrentStore) Name() string {
	return driverName
}

func (s *TorrentStore) Update(torrent store.Torrent) error {
	const q = `
		UPDATE 
		    torrent 
		SET
		    info_hash = ?,
		    total_completed = ?,
		    total_uploaded = ?,
		    total_downloaded = ?,
		    is_deleted = ?,
		    is_enabled = ?,
		    reason = ?,
		    multi_up = ?,
		    multi_dn = ?,
		    announces = ?
		WHERE
			info_hash = ?
			`
	_, err := s.db.Exec(q,
		torrent.InfoHash.Bytes(),
		torrent.Snatches,
		torrent.Uploaded,
		torrent.Downloaded,
		torrent.IsDeleted,
		torrent.IsEnabled,
		torrent.Reason,
		torrent.MultiUp,
		torrent.MultiDn,
		torrent.Announces,
		torrent.InfoHash.Bytes())
	if err != nil {
		return errors.Wrap(err, "Failed to update torrent")
	}
	return nil
}

// Sync batch updates the backing store with the new TorrentStats provided
func (s *TorrentStore) Sync(b map[store.InfoHash]store.TorrentStats) error {
	const q = `CALL torrent_update_stats(?, ?, ?, ?, ?, ?, ?)`
	tx, err := s.db.Begin()
	if err != nil {
		return errors.Wrap(err, "Failed to being torrent Sync() tx")
	}
	stmt, err2 := tx.Prepare(q)
	if err2 != nil {
		return errors.Wrap(err2, "Failed to prepare torrent Sync() tx")
	}
	for ih, stats := range b {
		if _, err := stmt.Exec(
			ih.Bytes(),
			stats.Downloaded,
			stats.Uploaded,
			stats.Announces,
			stats.Snatches,
			stats.Seeders,
			stats.Leechers,
		); err != nil {
			if err := tx.Rollback(); err != nil {
				log.Errorf("Failed to roll back torrent Sync() tx")
			}
			return errors.Wrap(err, "Failed to exec torrent Sync() tx")
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "Failed to commit torrent Sync() tx")
	}
	return nil
}

// Conn returns the underlying database driver
func (s *TorrentStore) Conn() interface{} {
	return s.db
}

// WhiteListDelete removes a client from the global whitelist
func (s *TorrentStore) WhiteListDelete(client store.WhiteListClient) error {
	const q = `CALL whitelist_delete_by_prefix(?)`
	if _, err := s.db.Exec(q, client.ClientPrefix); err != nil {
		return errors.Wrap(err, "Failed to delete client whitelist")
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (s *TorrentStore) WhiteListAdd(client store.WhiteListClient) error {
	const q = `CALL whitelist_add(?, ?)`
	if _, err := s.db.Exec(q, client.ClientPrefix, client.ClientName); err != nil {
		return errors.Wrap(err, "Failed to insert new whitelist entry")
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (s *TorrentStore) WhiteListGetAll() ([]store.WhiteListClient, error) {
	var wl []store.WhiteListClient
	const q = `CALL whitelist_all()`
	if err := s.db.Select(&wl, q); err != nil {
		return nil, errors.Wrap(err, "Failed to select client whitelists")
	}
	return wl, nil
}

// Close will close the underlying mysql database connection
func (s *TorrentStore) Close() error {
	return s.db.Close()
}

// Get returns a torrent for the hash provided
func (s *TorrentStore) Get(t *store.Torrent, hash store.InfoHash, deletedOk bool) error {
	const q = `CALL torrent_by_infohash(?, ?)`
	err := s.db.Get(t, q, hash.Bytes(), deletedOk)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return consts.ErrInvalidInfoHash
		}
		return err
	}
	if t.IsDeleted && !deletedOk {
		return consts.ErrInvalidInfoHash
	}
	return nil
}

// Add inserts a new torrent into the backing store
func (s *TorrentStore) Add(t store.Torrent) error {
	const q = `CALL torrent_add(?)`
	_, err := s.db.Exec(q, t.InfoHash.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (s *TorrentStore) Delete(ih store.InfoHash, dropRow bool) error {
	var err error
	if dropRow {
		const dropQ = `CALL torrent_delete(?)`
		_, err = s.db.Exec(dropQ, ih.Bytes())
	} else {
		const updateQ = `CALL torrent_disable(?)`
		_, err = s.db.NamedExec(updateQ, ih.Bytes())

	}
	if err != nil {
		return err
	}
	return nil
}

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
	defer func() {
		if err := rows.Close(); err != nil {
			log.Errorf("failed to close query rows: %s", err)
		}
	}()
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

var (
	connections   map[string]*sqlx.DB
	connectionsMu *sync.RWMutex
)

func getOrCreateConn(cfg *config.StoreConfig) (*sqlx.DB, error) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	existing, found := connections[cfg.Host]
	if found {
		return existing, nil
	}
	db, err := sqlx.Connect(driverName, cfg.DSN())
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to mysql database")
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(time.Second * 10)
	connections[cfg.Host] = db
	return db, nil
}

type userDriver struct{}

// New creates a new mysql backed user store.
func (ud userDriver) New(cfg interface{}) (store.UserStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db, err := getOrCreateConn(c)
	if err != nil {
		return nil, err
	}
	return &UserStore{db: db}, nil
}

type torrentDriver struct{}

// New initialize a TorrentStore implementation using the mysql backing store
func (td torrentDriver) New(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db, err := getOrCreateConn(c)
	if err != nil {
		return nil, err
	}
	return &TorrentStore{db: db}, nil
}

type peerDriver struct{}

// New returns a mysql backed store.PeerStore driver
func (pd peerDriver) New(cfg interface{}) (store.PeerStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db, err := getOrCreateConn(c)
	if err != nil {
		return nil, err
	}
	return &PeerStore{db: db}, nil
}

func init() {
	connections = make(map[string]*sqlx.DB)
	connectionsMu = &sync.RWMutex{}
	store.AddUserDriver(driverName, userDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
	store.AddPeerDriver(driverName, peerDriver{})
}
