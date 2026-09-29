package goup

import "testing"

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
