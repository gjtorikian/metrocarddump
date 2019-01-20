package main

import (
	"fmt"
	"os"

	"github.com/gjtorikian/metrocarddump/metrocarddump"
)

func main() {
	app := metrocarddump.NewApp()

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
