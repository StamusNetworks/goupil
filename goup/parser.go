package goup

import (
	"strings"

	"github.com/StamusNetworks/goupil/set"
	"golang.org/x/net/idna"
)

type Parser struct {
	Suffixes  set.Items[string]
	Windcards set.Items[string]
	Negate    set.Items[string]

	start  int
	offset int
	last   int
	dot    int
}

func (p *Parser) Parse(str string) *HostnameInfo {
	defer p.reset()
	if str == "" {
		return nil
	}

	hni := &HostnameInfo{}
	stats := &Stats{}
	hni.URL = str

	parts := stripURL(str)
	hni.Scheme, hni.ResourcePath, hni.QueryString = parts.scheme, parts.resourcePath, parts.queryString

	str = parts.host
	if isIPv6(str) || isIPv4(str) {
		return fallback(hni, stats, str)
	}
	if strings.HasSuffix(str, ".") {
		if l := len(str); l > 0 {
			str = str[:l-1]
		}
	}
	p.dot = strings.Index(str, ".")

	for p.dot > 0 {
		stats.HostTokenCount++
		if p.dot > stats.HostMaxTokenLength {
			stats.HostMaxTokenLength = p.dot
		}
		p.offset = p.start + p.dot + 1
		if p.Suffixes.Has(str[p.offset:]) || (p.Windcards.Has(str[p.offset:]) && p.Negate.Has(str[p.start:])) {
			stats.HostMaxTokenLength = max(stats.HostMaxTokenLength, len(str)-p.offset)
			stats.HostTokenCount++
			parts := p.fillMatch(str, p.offset, p.start)
			hni.Host, hni.TLD, hni.Domain, hni.Subdomain, hni.DomainWithoutTld =
				parts.host, parts.tld, parts.domain, parts.subdomain, parts.domainWithoutTld
			return finish(hni, stats, false)
		}

		if p.Windcards.Has(str[p.offset:]) {
			stats.HostMaxTokenLength = max(stats.HostMaxTokenLength, len(str)-p.start)
			stats.HostTokenCount++
			parts := p.fillMatch(str, p.start, p.last)
			hni.Host, hni.TLD, hni.Domain, hni.Subdomain, hni.DomainWithoutTld =
				parts.host, parts.tld, parts.domain, parts.subdomain, parts.domainWithoutTld
			return finish(hni, stats, false)
		}

		p.last = p.start
		p.start = p.offset
		p.dot = strings.Index(str[p.offset:], ".")
	}
	return fallback(hni, stats, str)
}

func (p *Parser) reset() {
	p.start = 0
	p.offset = 0
	p.last = 0
	p.dot = 0
}

func (p *Parser) fillMatch(str string, hi, lo int) domainParts {
	parts := domainParts{host: str, tld: str[hi:], domain: str[lo:]}
	if lo > 0 {
		parts.subdomain = str[:lo-1]
	}
	if hi > 0 {
		parts.domainWithoutTld = str[lo : hi-1]
	}
	return parts
}

func NewParser(suffixes []string) (*Parser, error) {
	p := &Parser{
		Suffixes:  set.NewItems[string](),
		Windcards: set.NewItems[string](),
		Negate:    set.NewItems[string](),
	}
	for _, line := range suffixes {
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		ascii, err := idna.ToASCII(line)
		if err != nil {
			return nil, err
		}
		if ascii != line {
			if err := addLineToDomains(p, ascii); err != nil {
				return nil, err
			}
		}
		if err := addLineToDomains(p, line); err != nil {
			return nil, err
		}
	}
	return p, nil
}
