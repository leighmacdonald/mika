package consts

import "time"

// VersionHash is replaced with git version hash at build time
var (
	BuildVersion = "master"
	BuildTime    = time.Now().UTC().Format(time.ANSIC)
)
