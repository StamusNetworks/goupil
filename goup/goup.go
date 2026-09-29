package goup

import (
	"fmt"
	"strings"
)

type HostnameInfo struct {
	Scheme           string `json:"scheme,omitempty"`
	ResourcePath     string `json:"resource_path,omitempty"`
	Host             string `json:"host"`
	URL              string `json:"url"`
	TLD              string `json:"tld"`
	Domain           string `json:"domain"`
	Subdomain        string `json:"subdomain"`
	DomainWithoutTld string `json:"domain_without_tld"`
	QueryString      string `json:"query_string,omitempty"`
	Stats            *Stats `json:"stats,omitempty"`
}

type Stats struct {
	DomainLength       int  `json:"domain_length"`
	SubdomainLength    int  `json:"subdomain_length"`
	HostLength         int  `json:"host_length"`
	HostTokenCount     int  `json:"host_token_count"`
	HostMaxTokenLength int  `json:"host_max_token_length"`
	Failed             bool `json:"failed"`
}

type domainParts struct {
	host, tld, domain, subdomain, domainWithoutTld string
}

type urlParts struct {
	host, scheme, resourcePath, queryString string
}

func stripURL(str string) urlParts {
	schemeEnd := strings.Index(str, "://")
	if schemeEnd == -1 {
		return urlParts{host: str}
	}
	parts := urlParts{scheme: str[:schemeEnd]}
	str = str[schemeEnd+3:]

	pathStart := strings.Index(str, "/")
	if pathStart == -1 {
		parts.host = str
		return parts
	}
	parts.resourcePath = str[pathStart:]
	str = str[:pathStart]

	if queryStart := strings.Index(parts.resourcePath, "?"); queryStart > -1 {
		parts.queryString = parts.resourcePath[queryStart:]
		parts.resourcePath = parts.resourcePath[:queryStart]
	}

	if portSep := strings.LastIndex(str, ":"); portSep > -1 {
		if bracketEnd := strings.Index(str, "]"); bracketEnd == -1 || bracketEnd < portSep {
			str = str[:portSep]
		}
	}
	parts.host = str
	return parts
}

func fallback(hni *HostnameInfo, stats *Stats, str string) *HostnameInfo {
	if hni.URL == "" {
		hni.URL = str
	}
	hni.Host = str
	hni.Domain = str
	hni.DomainWithoutTld = str
	return finish(hni, stats, true)
}

func finish(hni *HostnameInfo, stats *Stats, failed bool) *HostnameInfo {
	stats.Failed = failed
	stats.DomainLength = len(hni.Domain)
	stats.SubdomainLength = len(hni.Subdomain)
	stats.HostLength = len(hni.Host)
	hni.Stats = stats
	return hni
}

func isIPv6(host string) bool {
	if len(host) < 3 {
		return false
	}
	return strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]")
}

func isIPv4(host string) bool {
	if len(host) < 7 || len(host) > 15 {
		return false
	}
	var digits, dots int
	for i := range host {
		switch ch := host[i]; {
		case ch == '.':
			if digits < 1 || digits > 3 {
				return false
			}
			digits = 0
			dots++
		case ch >= '0' && ch <= '9':
			digits++
		default:
			return false
		}
	}
	if dots == 3 && (digits > 0 && digits <= 3) {
		return true
	}
	return false
}

func addLineToDomains(p *Parser, line string) error {
	if strings.HasPrefix(line, "*") {
		if bits := strings.SplitN(line, ".", 2); len(bits) == 2 {
			p.Windcards.Add(bits[1])
		} else {
			return fmt.Errorf("invalid domain: %s", line)
		}
	} else if strings.HasPrefix(line, "!") && len(line) > 2 {
		p.Negate.Add(line[1:])
	} else {
		p.Suffixes.Add(line)
	}
	return nil
}
