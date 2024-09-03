package mate

import (
	"encoding/json"
	"errors"
	"github.com/TomasBorquez/logger"
)

func HeaderParser(requestBuffer []byte) Request {
	var headers []string
	var lastHeader = 0
	req := Request{
		Method:    "",
		Path:      "",
		Host:      "",
		UserAgent: "",
		Accept:    "",
	}

	// Split headers in strings
	for i, b := range requestBuffer {
		char := string(b)

		if char == "\n" {
			header := ""

			for j := lastHeader; j < i-1; j++ {
				header += string(requestBuffer[j])
			}

			lastHeader = i + 1
			headers = append(headers, header)
		}
	}

	// Parse first line
	var lastSpace = 0
	var firstLine = headers[0]
	for i := 0; i < len(firstLine); i++ {
		char := string(firstLine[i])
		if char == " " {
			if req.Method == "" {
				for j := lastSpace; j < len(firstLine); j++ {
					char1 := string(firstLine[j])
					if char1 == " " {
						break
					}
					req.Method += char1
				}
			} else if req.Path == "" {
				for j := lastSpace; j < len(firstLine); j++ {
					char1 := string(firstLine[j])
					if char1 == " " {
						break
					}
					req.Path += char1
				}
				break
			}
			lastSpace = i + 1
		}
	}

	// Parse rest of headers
	for i := 1; i < len(headers); i++ {
		header := headers[i]
		for j, c := range header {
			char := string(c)
			if char == " " {
				name := ""
				value := ""
				for k := 0; k < len(header); k++ {
					currChar := string(header[k])
					if currChar == ":" {
						continue
					}

					if k < j {
						name += currChar
					} else if k > j {
						value += currChar
					}
				}
				SelectProperty(name, value, &req)
			}
		}
	}

	return req
}

func SelectProperty(name string, value string, req *Request) {
	switch name {
	case "Host":
		req.Host = value
	case "host":
		req.Host = value
	case "User-Agent":
		req.UserAgent = value
	case "Accept":
		req.Accept = value
	case "Content-Type":
		req.ContentType = value
	case "Authorization":
		req.Authorization = value
	case "Accept-Encoding":
		req.AcceptEncoding = value
	case "Content-Length":
		req.ContentLength = value
	case "Referer":
		req.Referer = value
	case "Cookie":
		req.Cookie = value
	case "Origin":
		req.Origin = value
	case "Cache-Control":
		req.CacheControl = value
	case "X-Forwarded-For":
		req.XForwardedFor = value
	case "X-Requested-With":
		req.XRequestedWith = value
	case "Connection":
		req.Connection = value
	default:
		logger.Warning("[MATE]: Could not find property: %s with the value %s", name, value)
	}
}

func (c *Context) Status(code int) *Context {
	c.Res.statusCode = code
	return c
}

func (c *Context) SendString(response string) error {
	c.Res.ContentType = "text/plain"
	c.Res.Body = response
	return nil
}

func (c *Context) JSON(data any) error {
	c.Res.ContentType = "application/json"
	b, _ := json.Marshal(data)
	c.Res.Body = string(b)
	return nil
}

func (c *Context) HTML(response string) error {
	c.Res.ContentType = "text/html"
	c.Res.Body = response
	return nil
}

func (c *Context) Params(key string) (value string, err error) {
	if value, exists := c.params[key]; exists {
		return value, nil
	} else {
		return "", errors.New("property " + key + " does not exist")
	}
}
