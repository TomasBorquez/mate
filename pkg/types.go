package mate

import "time"

type Context struct {
	Req    Request
	Res    Response
	params map[string]string
	time   time.Time
}

type Request struct {
	Method         string
	Path           string
	Host           string
	UserAgent      string
	Accept         string
	ContentType    string
	Authorization  string
	AcceptEncoding string
	ContentLength  string
	Referer        string
	Cookie         string
	Origin         string
	CacheControl   string
	XForwardedFor  string
	XRequestedWith string
	Connection     string
	body           string
}

type Response struct {
	statusCode  int
	ContentType string
	Body        string
}

type Configuration struct {
	// RequestTimeout in seconds
	RequestTimeout int
	// MinimumTransferSpeed Bytes per second
	MinimumTransferSpeed int
	MaxContentLength     int
	Logging              bool
}

var DefaultConfiguration = Configuration{
	RequestTimeout:       30,
	MinimumTransferSpeed: 100,
	MaxContentLength:     10 * 1024 * 1024,
	Logging:              false,
}

type Application struct {
	config Configuration
}
