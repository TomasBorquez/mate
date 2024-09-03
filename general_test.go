package mate

import (
	"github.com/stretchr/testify/assert"
	"http-server/internal"
	mate "http-server/pkg"
	"log"
	"net/http"
	"testing"
	"time"
)

func runServer() {
	serverReady := make(chan bool)
	go func() {
		app := mate.New(mate.Configuration{
			Logging: true,
		})
		
		app.Get("/", func(ctx *mate.Context) error {
			return ctx.SendString("Test String")
		})
		
		app.ListenNotify("8080", serverReady)
	}()
	
	select {
	case <-serverReady:
		// Server is ready, continue with the test
		return
	case <-time.After(5 * time.Second):
		log.Fatal("Server didn't start within the timeout period")
	}
}

func TestSimpleGetRequest(t *testing.T) {
	t.Parallel()
	runServer()
	
	t.Run("returns 200 test string", func(t *testing.T) {
		res, err := http.Get(internal.BaseUrl + "/")
		if err != nil {
			log.Fatal(err)
			return
		}
		
		assert.Equal(t, res.Status, "200 OK")
		assert.Equal(t, internal.ReadResponseBodyString(res.Body), "Test String")
	})
	
	t.Run("returns 404 empty string", func(t *testing.T) {
		res, err := http.Get(internal.BaseUrl + "/404")
		if err != nil {
			log.Fatal(err)
			return
		}
		
		assert.Equal(t, res.Status, "404 Not Found")
		assert.Equal(t, internal.ReadResponseBodyString(res.Body), "")
	})
	
	t.Run("returns 500 when no body is sent", func(t *testing.T) {
		type PostBody struct {
			Title string `json:"title"`
		}
		
		reader := internal.TransformBody(PostBody{
			Title: "test",
		})
		
		res, err := http.Post(internal.BaseUrl+"/", "application/json", reader)
		if err != nil {
			log.Fatal(err)
			return
		}
		
		assert.Equal(t, res.Status, "500 Internal Server Error")
		assert.Equal(t, internal.ReadResponseBodyString(res.Body), "")
	})
}
