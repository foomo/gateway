package sitemap

import (
	"encoding/xml"
	"fmt"
	"net/http"
)

type index struct {
	XMLName  xml.Name `xml:"sitemapindex"`
	XMLNS    string   `xml:"xmlns,attr"`
	Sitemaps []loc    `xml:"sitemap"`
}

type loc struct {
	Loc string `xml:"loc"`
}

// Serve writes a sitemapindex XML document containing the given URLs to w.
func Serve(w http.ResponseWriter, urls []string) {
	idx := index{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for _, u := range urls {
		idx.Sitemaps = append(idx.Sitemaps, loc{Loc: u})
	}

	w.Header().Set("Content-Type", "application/xml")
	_, _ = fmt.Fprint(w, xml.Header)

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(idx)
}
