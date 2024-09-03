package mate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TomasBorquez/logger"
	"io"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Callback = func(ctx *Context) error

type Handler struct {
	extract func(path string) map[string]string
	path    *regexp.Regexp
	cb      Callback
}

var getHandlers []Handler
var postHandlers []Handler
var putHandlers []Handler
var deleteHandlers []Handler
var errorHandler func(ctx *Context, err error) error
var notFoundHandler Callback

func New(config ...Configuration) Application {
	app := Application{
		config: DefaultConfiguration,
	}
	
	if len(config) > 0 {
		app.config = config[0]
		
		if app.config.RequestTimeout == 0 {
			app.config.RequestTimeout = DefaultConfiguration.RequestTimeout
		}
		
		if app.config.MinimumTransferSpeed == 0 {
			app.config.MinimumTransferSpeed = DefaultConfiguration.MinimumTransferSpeed
		}
		
		if app.config.MaxContentLength == 0 {
			app.config.MaxContentLength = DefaultConfiguration.MaxContentLength
		}
	}
	
	return app
}

func (app Application) Listen(port string) Application {
	listen, err := net.Listen("tcp", ":"+port)
	
	if err != nil {
		logger.Error("Error meanwhile listening: %v", err)
		return app
	}
	
	defer listen.Close()
	
	logger.Success("[MATE]: Listening on port: " + port)
	
	for {
		connection, err := listen.Accept()
		if err != nil {
			fmt.Println(err)
		}
		
		go HandleRequest(connection, &app.config)
	}
}

func (app Application) ListenNotify(port string, ready chan bool) Application {
	listen, err := net.Listen("tcp", ":"+port)
	
	if err != nil {
		ready <- false
		logger.Error("Error meanwhile listening: %v", err)
		return app
	}
	
	defer listen.Close()
	
	ready <- true
	logger.Success("[MATE]: Listening on port: " + port)
	
	for {
		connection, err := listen.Accept()
		if err != nil {
			fmt.Println(err)
		}
		
		go HandleRequest(connection, &app.config)
	}
}

func HandleRequest(connection net.Conn, config *Configuration) {
	defer connection.Close()
	
	ctx := Context{
		Res: Response{
			statusCode: 200,
			Body:       "",
		},
		time: time.Now(),
	}
	
	deadline := time.Now().Add(time.Duration(config.RequestTimeout) * time.Second)
	err := connection.SetDeadline(deadline)
	
	if err != nil {
		logger.Error("[MATE]: Error setting config.RequestTimeout: %v", err)
		return
	}
	
	headerBuffer := make([]byte, 8192)
	n, err := connection.Read(headerBuffer)
	if err != nil {
		logger.Error("[MATE]: Error reading connection: %v", err)
		sendResponse(connection, &ctx, 400, config)
		return
	}
	
	ctx.Req = HeaderParser(headerBuffer[:n])
	
	contentLength, err := strconv.Atoi(ctx.Req.ContentLength)
	
	if contentLength > config.MaxContentLength {
		logger.Error("[MATE]: Content length exceeds maximum allowed")
		sendResponse(connection, &ctx, 413, config)
		return
	}
	
	if contentLength > 0 {
		fullBuffer, err := readWithSpeedCheck(connection, contentLength, config)
		if err != nil {
			logger.Error("[MATE]: %v", err)
			if err.Error() == "transfer speed too slow" {
				sendResponse(connection, &ctx, 408, config)
				return
			}
			sendResponse(connection, &ctx, 500, config)
			return
		}
		
		ctx.Req.body = string(fullBuffer)
		fmt.Println(string(fullBuffer))
	}
	fmt.Println(string(headerBuffer))
	
	handler, err := ResolveHandler(&ctx)
	if err != nil {
		if notFoundHandler == nil {
			logger.Error("[MATE]: 404 Not found, set a notFoundHandler with `app.setNotFound()`")
			sendResponse(connection, &ctx, 404, config)
			return
		}
		
		err = notFoundHandler(&ctx)
		if err != nil {
			errorHandler(&ctx, err)
		}
		sendResponse(connection, &ctx, ctx.Res.statusCode, config)
		return
	}
	ctx.params = handler.extract(ctx.Req.Path)
	err = handler.cb(&ctx)
	if err != nil {
		errorHandler(&ctx, err)
	}
	
	sendResponse(connection, &ctx, ctx.Res.statusCode, config)
}

type readResult struct {
	n    int
	err  error
}

func readWithTimeout(conn io.Reader, buffer []byte) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	ch := make(chan readResult)
	go func() {
		n, err := conn.Read(buffer)
		ch <- readResult{n: n, err: err}
	}()
	
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case res := <-ch:
		return res.n, res.err
	}
}

func readWithSpeedCheck(connection io.Reader, contentLength int, config *Configuration) ([]byte, error) {
	fullBuffer := make([]byte, 0, contentLength)
	buffer := make([]byte, 8192) // 8KB buffer
	
	startTime := time.Now()
	totalBytesRead := 0
	lastCheckTime := startTime
	lastCheckBytes := 0
	
	for totalBytesRead < contentLength {
		n, err := readWithTimeout(connection, buffer)
		
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading request: %v", err)
		}
		
		fullBuffer = append(fullBuffer, buffer[:n]...)
		totalBytesRead += n
		
		// Check speed after grace period (5 seconds or 10% of data, whichever comes first)
		if time.Since(startTime) > 5*time.Second || totalBytesRead > contentLength/10 {
			elapsedSinceLastCheck := time.Since(lastCheckTime)
			bytesSinceLastCheck := totalBytesRead - lastCheckBytes
			
			if elapsedSinceLastCheck > time.Second { // Update at most once per second
				bytesPerSecond := float64(bytesSinceLastCheck) / elapsedSinceLastCheck.Seconds()
				if int(bytesPerSecond) < config.MinimumTransferSpeed {
					return nil, fmt.Errorf("transfer speed too slow")
				}
				
				lastCheckTime = time.Now()
				lastCheckBytes = totalBytesRead
			}
		}
		
		if totalBytesRead >= contentLength {
			break
		}
	}
	
	return fullBuffer, nil
}

func sendResponse(connection net.Conn, ctx *Context, statusCode int, config *Configuration) {
	code, err := StringifyCode(statusCode)
	if err != nil {
		logger.Error("[MATE]: %v", err)
		return
	}
	
	if config.Logging {
		var color string
		switch {
		case ctx.Res.statusCode < 300:
			color = logger.Green
		case ctx.Res.statusCode < 400:
			color = logger.Blue
		case ctx.Res.statusCode < 500:
			color = logger.Orange
		default:
			color = logger.Red
		}
		
		logger.Custom("[MATE]: %s%d%s - %dms | %s %s",
			color,
			ctx.Res.statusCode,
			logger.Reset,
			time.Since(ctx.time).Milliseconds(),
			ctx.Req.Method,
			ctx.Req.Path)
	}
	
	response := fmt.Sprintf("HTTP/1.1 %s\r\n"+
		"Content-Type: %s\r\n"+
		"Content-Length: %d\r\n"+
		"\r\n"+
		"%s",
		code, ctx.Res.ContentType, len(ctx.Res.Body), ctx.Res.Body)
	
	_, err = connection.Write([]byte(response))
	if err != nil {
		logger.Error("[MATE]: Error writing response: %v", err)
		return
	}
}

func (app Application) Get(path string, cb Callback) Application {
	regExpPath, extract, err := PathToRegexp(path)
	
	if err != nil {
		logger.Error("Error meanwhile parsing path to RegExp, for path: " + path)
		return app
	}
	
	handler := Handler{
		extract: extract,
		path:    regExpPath,
		cb:      cb,
	}
	
	getHandlers = append(getHandlers, handler)
	return app
}

func (app Application) Post(path string, cb Callback) Application {
	regExpPath, extract, err := PathToRegexp(path)
	
	if err != nil {
		logger.Error("[MATE]: Error meanwhile parsing path to RegExp, for path: %s", path)
		return app
	}
	
	handler := Handler{
		extract: extract,
		path:    regExpPath,
		cb:      cb,
	}
	
	postHandlers = append(postHandlers, handler)
	return app
}

func (app Application) Put(path string, cb Callback) Application {
	regExpPath, extract, err := PathToRegexp(path)
	
	if err != nil {
		logger.Error("[MATE]: Error meanwhile parsing path to RegExp, for path: %s", path)
		return app
	}
	
	handler := Handler{
		extract: extract,
		path:    regExpPath,
		cb:      cb,
	}
	
	putHandlers = append(putHandlers, handler)
	return app
}

func (app Application) Delete(path string, cb Callback) Application {
	regExpPath, extract, err := PathToRegexp(path)
	
	if err != nil {
		logger.Error("[MATE]: Error meanwhile parsing path to RegExp, for path: %s", path)
		return app
	}
	
	handler := Handler{
		extract: extract,
		path:    regExpPath,
		cb:      cb,
	}
	
	deleteHandlers = append(deleteHandlers, handler)
	return app
}

func (app Application) SetNotFound(cb Callback) Application {
	notFoundHandler = cb
	return app
}

func (app Application) SetError(cb func(ctx *Context, err error) error) Application {
	errorHandler = cb
	return app
}

func (c *Context) ParseBody(out interface{}) error {
	body := strings.TrimRight(c.Req.body, "\x00")
	
	// Ensure out is a pointer
	if reflect.ValueOf(out).Kind() != reflect.Ptr {
		return fmt.Errorf("ParseBody requires a pointer, got %v", reflect.TypeOf(out))
	}
	
	if err := json.Unmarshal([]byte(body), out); err != nil {
		return err
	}
	
	return nil
}

func ResolveHandler(ctx *Context) (handler *Handler, error error) {
	var handlers []Handler
	temp := strings.Split(ctx.Req.Path, ":")[0]
	ctx.Req.Path = temp
	
	switch ctx.Req.Method {
	case "GET":
		handlers = getHandlers
	case "PUT":
		handlers = putHandlers
	case "POST":
		handlers = postHandlers
	case "DELETE":
		handlers = deleteHandlers
	}
	
	for _, handler := range handlers {
		didMatch := handler.path.MatchString(ctx.Req.Path)
		
		if !didMatch {
			continue
		}
		
		return &handler, nil
	}
	
	return nil, errors.New("could not find a handler")
}

func StringifyCode(code int) (codeString string, error error) {
	switch code {
	// 2xx Success
	case 200:
		return "200 OK", nil
	case 201:
		return "201 Created", nil
	case 202:
		return "202 Accepted", nil
	case 204:
		return "204 No Content", nil
	case 206:
		return "206 Partial Content", nil
	
	// 3xx Redirection
	case 300:
		return "300 Multiple Choices", nil
	case 301:
		return "301 Moved Permanently", nil
	case 302:
		return "302 Found", nil
	case 303:
		return "303 See Other", nil
	case 304:
		return "304 Not Modified", nil
	case 307:
		return "307 Temporary Redirect", nil
	case 308:
		return "308 Permanent Redirect", nil
	
	// 4xx Client Errors
	case 400:
		return "400 Bad Request", nil
	case 401:
		return "401 Unauthorized", nil
	case 402:
		return "402 Payment Required", nil
	case 403:
		return "403 Forbidden", nil
	case 404:
		return "404 Not Found", nil
	case 405:
		return "405 Method Not Allowed", nil
	case 406:
		return "406 Not Acceptable", nil
	case 408:
		return "408 Request Timeout", nil
	case 409:
		return "409 Conflict", nil
	case 410:
		return "410 Gone", nil
	case 411:
		return "411 Length Required", nil
	case 413:
		return "413 Payload Too Large", nil
	case 414:
		return "414 URI Too Long", nil
	case 415:
		return "415 Unsupported Media Type", nil
	case 418:
		return "418 I'm a teapot", nil
	case 429:
		return "429 Too Many Requests", nil
	
	// 5xx Server Errors
	case 500:
		return "500 Internal Server Error", nil
	case 501:
		return "501 Not Implemented", nil
	case 502:
		return "502 Bad Gateway", nil
	case 503:
		return "503 Service Unavailable", nil
	case 504:
		return "504 Gateway Timeout", nil
	case 505:
		return "505 HTTP Version Not Supported", nil
	default:
		return "", errors.New("statusCode " + strconv.Itoa(code) + " does not exist in HTTP")
	}
}
