package main

import (
	"github.com/TomasBorquez/logger"
	"http-server/pkg"
)

func main() {
	app := mate.New(mate.Configuration{
		Logging:      true,
	})
	
	app.Get("/", func(ctx *mate.Context) error {
		return ctx.SendString("Test String")
	})

	app.SetNotFound(func(ctx *mate.Context) error {
		return ctx.Status(404).HTML("<h1>Not Found</h1>")
	})

	app.SetError(func(ctx *mate.Context, err error) error {
		logger.Error("%v", err)
		return ctx.JSON(map[string]bool{"success": false})
	})

	app.Listen("3000")
}
