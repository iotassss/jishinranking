package domain

import "encoding/xml"

// Atom feed (JMA eqvol.xml)
type Feed struct {
	XMLName  xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
	Lang     string   `xml:"lang,attr,omitempty"` // xml:lang (JMAはlang="ja"として出してくることが多い)
	Title    string   `xml:"title"`
	Subtitle string   `xml:"subtitle,omitempty"`
	Updated  string   `xml:"updated"` // e.g. 2025-12-21T00:32:22+09:00
	ID       string   `xml:"id"`

	Links   []Link  `xml:"link"`
	Rights  *Rights `xml:"rights,omitempty"`
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Title   string `xml:"title"`
	ID      string `xml:"id"`
	Updated string `xml:"updated"` // e.g. 2025-12-20T15:32:10Z

	Author  *Author `xml:"author,omitempty"`
	Link    Link    `xml:"link,omitempty"`
	Content Content `xml:"content"`
}

type Author struct {
	Name string `xml:"name"`
}

type Link struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr,omitempty"`
}

type Rights struct {
	Type     string `xml:"type,attr,omitempty"`
	InnerXML string `xml:",innerxml"` // CDATA含めて中身を保持したい場合
}

type Content struct {
	Type string `xml:"type,attr,omitempty"` // text / html etc
	Text string `xml:",chardata"`
}
