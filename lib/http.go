package ircb

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func (c *Connection) HandleLinks(irc *IRC) {
	defer c.Log.Println("done handling link")
	i := strings.Index(irc.Message, "http")
	if i == -1 {

		return
	}
	s := irc.Message[i:]
	ss := strings.Split(s, " ")[0]
	_, err := url.Parse(ss)
	if err != nil {
		c.Log.Println("error parsing url:", err)
		return
	}

	if strings.Contains(ss, "localhost") || strings.Contains(ss, "::") {
		c.Log.Println("bad url:", ss)
		return
	}
	req, err := http.NewRequest(http.MethodGet, ss, nil)
	if err != nil {
		c.Log.Println("error making request:", err)
		return
	}

	c.Log.Println("sending request:", ss)
	t1 := time.Now()
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		c.Log.Println("error getting url:", ss, err)
		return
	}
	if resp.StatusCode != 200 {
		// reply error
		irc.Reply(c, fmt.Sprintf("%s %s", resp.Status, time.Now().Sub(t1)))

		return
	}
	defer resp.Body.Close()
	reader := io.LimitReader(resp.Body, 1024)
	meta := GetLinkTitleFromHTML(reader)
	// reply
	if meta.Title != "" {
		irc.Reply(c, fmt.Sprintf("%s %s %q", resp.Status, time.Now().Sub(t1), meta.Title))
		return
	}
	irc.Reply(c, fmt.Sprintf("%s %s", resp.Status, time.Now().Sub(t1)))
	return

}

type htmlMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"site_name"`
}

func GetLinkTitleFromHTML(reader io.Reader) *htmlMeta {
	z := html.NewTokenizer(reader)

	titleFound := false

	hm := new(htmlMeta)

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return hm
		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()
			if t.Data == `body` {
				return hm
			}
			if t.Data == "title" {
				titleFound = true
			}
			if t.Data == "meta" {
				desc, ok := extractMetaProperty(t, "description")
				if ok {
					hm.Description = desc
				}

				ogTitle, ok := extractMetaProperty(t, "og:title")
				if ok {
					hm.Title = ogTitle
				}

				ogDesc, ok := extractMetaProperty(t, "og:description")
				if ok {
					hm.Description = ogDesc
				}

				ogImage, ok := extractMetaProperty(t, "og:image")
				if ok {
					hm.Image = ogImage
				}

				ogSiteName, ok := extractMetaProperty(t, "og:site_name")
				if ok {
					hm.SiteName = ogSiteName
				}
			}
		case html.TextToken:
			if titleFound {
				t := z.Token()
				hm.Title = t.Data
				titleFound = false
			}
		}
	}
	return hm
}

func extractMetaProperty(t html.Token, prop string) (content string, ok bool) {
	for _, attr := range t.Attr {
		if attr.Key == "property" && attr.Val == prop {
			ok = true
		}

		if attr.Key == "content" {
			content = attr.Val
		}
	}

	return
}
