package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
)

const BaseUrl = "http://localhost:8080"

func ReadResponseBodyString(Body io.ReadCloser) string {
	body, err := io.ReadAll(Body)
	if err != nil {
		log.Fatalln(err)
	}
	
	return string(body)
}

func TransformBody(body any) *bytes.Reader {
	bodyBytes, err := json.Marshal(&body)
	if err != nil {
		log.Fatal(err)
	}
	
	return bytes.NewReader(bodyBytes)
}
