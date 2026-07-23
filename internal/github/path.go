package github

import (
	"net/url"
	"strconv"
	"strings"
)

func apiPath(parts ...string) string {
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		escaped = append(escaped, url.PathEscape(part))
	}
	return strings.Join(escaped, "/")
}

func apiPathWithQuery(path string, values url.Values) string {
	if len(values) == 0 {
		return path
	}
	return path + "?" + values.Encode()
}

func query(values map[string]string) url.Values {
	q := url.Values{}
	for key, value := range values {
		if value != "" {
			q.Set(key, value)
		}
	}
	return q
}

func queryInt(values map[string]int) url.Values {
	q := url.Values{}
	for key, value := range values {
		q.Set(key, strconv.Itoa(value))
	}
	return q
}

func repoFullName(owner, repo string) string {
	return owner + "/" + repo
}
