package consts

// AnnounceType is valid announce event values
type AnnounceType string

// Announce types
const (
	STARTED   AnnounceType = "started"
	STOPPED   AnnounceType = "stopped"
	COMPLETED AnnounceType = "completed"
	// Partial seeders send paused event BEP-21
	PAUSED   AnnounceType = "paused"
	ANNOUNCE AnnounceType = ""
)

// ParseAnnounceType returns the AnnounceType from a string
func ParseAnnounceType(t string) AnnounceType {
	switch t {
	case "started":
		return STARTED
	case "stopped":
		return STOPPED
	case "completed":
		return COMPLETED
	case "paused":
		return PAUSED
	default:
		return ANNOUNCE
	}
}
