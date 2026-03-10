# Parallel HTTP implementation in Go

### Usage:
```go
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
```

## TODO
- [x] Parse requests
- [x] Handle routes
- [x] Set custom response code
- [x] Add tests
- [x] Add config struct to New
    - [x] Logging
    - [x] Timeout
    - [x] Minimum speed
    - [x] Cax content-length
    - [x] Set defaults
