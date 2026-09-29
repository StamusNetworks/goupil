package goup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var fuzzSeeds = []string{
	"www.example.com",
	"drive.google.com.",
	"a.b.example.co.uk",
	"foo.bar.ck",
	"www.ck",
	"8.8.8.8",
	"[fe80::12]",
	"https://www.example.com:8080/path/index.html?q=1",
	"http://[fe80::12]:443/",
	"123.高知.jp",
}

func FuzzParse(f *testing.F) {
	for _, item := range fuzzSeeds {
		f.Add(item)
	}

	filter, err := NewParser([]string{"com", "uk", "co.uk", "jp", "高知.jp", "*.ck", "!www.ck"})
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, str string) {
		filter.Parse(str)
	})
}

func TestParseNegation(t *testing.T) {
	p, err := NewParser([]string{"jp", "*.kawasaki.jp", "!city.kawasaki.jp", "*.ck", "!www.ck"})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		host, tld, domain, subdomain, domainWithoutTld string
	}{
		{host: "city.kawasaki.jp", tld: "kawasaki.jp", domain: "city.kawasaki.jp", domainWithoutTld: "city"},
		{host: "www.city.kawasaki.jp", tld: "kawasaki.jp", domain: "city.kawasaki.jp", subdomain: "www", domainWithoutTld: "city"},
		{host: "www.ck", tld: "ck", domain: "www.ck", domainWithoutTld: "www"},
		{host: "a.www.ck", tld: "ck", domain: "www.ck", subdomain: "a", domainWithoutTld: "www"},
		{host: "a.b.ck", tld: "b.ck", domain: "a.b.ck", domainWithoutTld: "a"},
		{host: "www.foo.kawasaki.jp", tld: "foo.kawasaki.jp", domain: "www.foo.kawasaki.jp", domainWithoutTld: "www"},
	}
	for _, test := range tests {
		t.Run(test.host, func(t *testing.T) {
			hni := p.Parse(test.host)
			assert.Equal(t, test.tld, hni.TLD)
			assert.Equal(t, test.domain, hni.Domain)
			assert.Equal(t, test.subdomain, hni.Subdomain)
			assert.Equal(t, test.domainWithoutTld, hni.DomainWithoutTld)
			assert.False(t, hni.Stats.Failed)
		})
	}
}
