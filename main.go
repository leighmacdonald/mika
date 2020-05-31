/*
Copyright © 2020 Leigh MacDonald <leigh.macdonald@gmail.com>

*/
package main

import (
	"github.com/leighmacdonald/mika/cmd"
	//_ "github.com/leighmacdonald/mika/store/http"
	_ "github.com/leighmacdonald/mika/store/memory"
	_ "github.com/leighmacdonald/mika/store/mysql"
	_ "github.com/leighmacdonald/mika/store/postgres"
	_ "github.com/leighmacdonald/mika/store/redis"
)

func main() {
	cmd.Execute()
}
