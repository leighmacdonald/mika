package memory

import (
	"github.com/leighmacdonald/mika/store"
	"testing"
)

func TestMemoryTorrentStore(t *testing.T) {
	store.TestStore(t, NewDriver())
}
