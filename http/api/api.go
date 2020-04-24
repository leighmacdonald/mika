// api package is used to remotely control all aspects of the tracker.
package api

import "mika/model"

type Api struct {
}

func TorrentDelete(ih model.InfoHash) error {
	return nil
}
