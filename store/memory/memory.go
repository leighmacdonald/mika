package memory

import (
	"mika/consts"
	"mika/model"
)

type Memory struct {
	torrents map[model.InfoHash]*model.Torrent
}

func (m *Memory) AddTorrent(t *model.Torrent) error {
	_, found := m.torrents[t.InfoHash]
	if found {
		return consts.ErrDuplicate
	}
	m.torrents[t.InfoHash] = t
	return nil
}

func New() *Memory {
	return &Memory{map[model.InfoHash]*model.Torrent{}}
}
