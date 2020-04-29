/*
Copyright © 2020 Leigh MacDonald <leigh.macdonald@gmail.com>

*/
package main

import (
	"mika/cmd"
	_ "mika/store/http"
	_ "mika/store/memory"
	_ "mika/store/mysql"
	_ "mika/store/redis"
)

func main() {
	cmd.Execute()
}
