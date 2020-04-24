// +build mysql

package mysql

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"mika/model"
)

type Store struct {
	db *sql.DB
}

func (s *Store) AddTorrent(t *model.Torrent) error {
	return nil
}

func (s *Store) DeleteTorrent(ih model.InfoHash) error {
	return nil
}

func (s *Store) AddPeer(p *model.Peer) error {
	return nil
}
func (s *Store) DeletePeer(p *model.Peer) error {
	return nil
}

func (s *Store) GetPeers(ih model.InfoHash) ([]*model.Peer, error) {
	return nil, nil
}

func (s *Store) GetScrape(ih model.InfoHash) {

}

func New(dsn string) *Store {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Panicf("Failed to connect to data store")
	}
	return &Store{
		db: db,
	}
}
