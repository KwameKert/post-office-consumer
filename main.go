package main

import (
	"office-consumer/app"
	"office-consumer/app/core"
)

func main() {
	config := core.NewConfig()
	app := &app.App{}
	app.Start(config)
}
