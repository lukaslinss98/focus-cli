package domain

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func Normalize(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("website is required")
	}

	isURL := strings.Contains(input, "://")
	candidate := input
	if !isURL {
		if strings.ContainsAny(input, "/?#@") {
			return "", fmt.Errorf("invalid website %q", input)
		}
		candidate = "https://" + input
	}

	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("invalid website %q", input)
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" || net.ParseIP(host) != nil || host == "localhost" || !validHostname(host) {
		return "", fmt.Errorf("invalid website %q", input)
	}

	if strings.HasPrefix(host, "www.") {
		host = strings.TrimPrefix(host, "www.")
	}
	if !strings.Contains(host, ".") {
		return "", fmt.Errorf("website must be a domain name")
	}
	return host, nil
}

func Variants(domain string) []string {
	return []string{domain, "www." + domain}
}

func validHostname(host string) bool {
	if len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if !((character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '-') {
				return false
			}
		}
	}
	return true
}
