package store

import "mika/model"

type DataStore interface {
	AddTorrent(t *model.Torrent) error
	DeleteTorrent(ih model.InfoHash) error
	AddPeer(p *model.Peer) error
	DeletePeer(p *model.Peer) error
	GetPeers(ih model.InfoHash) ([]*model.Peer, error)
	GetScrape(ih model.InfoHash)
}
