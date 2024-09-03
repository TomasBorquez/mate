package mate

import (
	"bytes"
	"context"
	"github.com/a-h/templ"
	"regexp"
	"strings"
)

func TemplToString(component templ.Component) (componentStringified string, renderError error) {
	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func PathToRegexp(path string) (*regexp.Regexp, func(string) map[string]string, error) {
	escaped := strings.Replace(regexp.QuoteMeta(path), `\:`, `:`, -1)

	pattern := regexp.MustCompile(`:(\w+)`).ReplaceAllString(escaped, `(?P<$1>[^/]+)`)

	pattern = "^" + pattern + "$"

	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, nil, err
	}

	extract := func(path string) map[string]string {
		match := r.FindStringSubmatch(path)
		results := make(map[string]string)
		for i, name := range r.SubexpNames() {
			if i != 0 && name != "" {
				results[name] = match[i]
			}
		}
		return results
	}

	return r, extract, nil
}
