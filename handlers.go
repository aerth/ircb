package ircb

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func verbIntHandler(c *Connection, irc *IRC) bool {
	verb, _ := strconv.Atoi(irc.Verb)
	switch verb {
	default: // unknown numerical verb
		c.Log.Printf("%s %q", irc.Verb, irc.Message)
		return handled
	case 331:
		c.Log.Printf("%s TOPIC: %q", irc.Raw, "")
		return handled
	case 353:
		c.Log.Printf("%s USER LIST: %q", irc.Raw, "")
		return handled
	case 372, 1, 2, 3, 4, 5, 6, 7, 0, 366:
		return handled
	case 221:
		c.Log.Printf("UMODE: %q", irc.Message)
		return handled
	case 433:

		c.Close()
		return handled
	}

}

func privmsgMasterHandler(c *Connection, irc *IRC) bool {
	if irc.ReplyTo != strings.Split(c.config.Master, ":")[0] {
		c.Log.Printf("not master: %s", irc.ReplyTo)
		return nothandled
	}

	if dur := time.Now().Sub(c.masterauth); dur > 5*time.Minute {
		c.Log.Println("need reauth after", dur)
		c.MasterCheck()
		return nothandled
	}
	c.Log.Println("got master message, parsing...")
	i := strings.Index(c.config.Master, ":")

	if i == -1 {
		c.Log.Println("bad config, not semicolon in Master field")
		return nothandled
	}
	if i >= len(c.config.Master) {
		c.Log.Println("bad config, bad semicolon in Master field")
		return nothandled
	}
	mp := c.config.Master[i+1:]
	if !strings.HasPrefix(irc.Message, mp) {

		// switch prefix
		if irc.Command == c.config.CommandPrefix && len(irc.Arguments) == 1 {
			c.config.CommandPrefix = irc.Arguments[0]
			c.Log.Printf("**New command prefix: %q", c.config.CommandPrefix)
			c.SendMaster("**New command prefix: %q", c.config.CommandPrefix)
			return handled
		}
		if c.config.Verbose {
			c.Log.Println("not master command prefixed")
		}
		return nothandled
	}
	irc.Message = strings.TrimPrefix(irc.Message, mp)

	irc.Command = strings.TrimSpace(strings.Split(irc.Message, " ")[0])
	args := strings.Split(strings.TrimPrefix(irc.Message, irc.Command), " ")
	for _, v := range args {
		if strings.TrimSpace(v) == "" {
			continue
		}
		irc.Arguments = append(irc.Arguments, v)
	}
	c.Log.Printf("master command request: %s, %v args)", irc.Command, len(irc.Arguments))
	if c.config.Verbose {
		c.Log.Printf("master command request: %s", irc)
	}
	if irc.Command != "" {
		c.Log.Println("trying master command:", irc.Command)
		if fn, ok := c.MasterMap[irc.Command]; ok {
			c.Log.Printf("master command found: %q", irc.Command)
			fn(c, irc)
			return handled
		}
	}
	c.Log.Printf("master command not found: %q", irc.Command)
	return nothandled

}

func privmsgHandler(c *Connection, irc *IRC) {

	// is karma
	if strings.HasPrefix(irc.To, "#") && c.parseKarma(irc.Message) {
		return

	}

	if irc.Command != "" {
		if fn, ok := c.CommandMap[irc.Command]; ok {
			c.Log.Printf("command found: %q", irc.Command)
			fn(c, irc)
			return
		}
	}

	// handle channel defined definitions
	if irc.Command != "" && len(irc.Arguments) == 0 {
		definition := c.getDefinition(irc.Command)
		if definition != "" {
			irc.Reply(c, definition)
			return
		}

	}

	if irc.Command != "" {
		c.Log.Printf("command not found: %q", irc.Command)
	}
	// try to parse http link title
	if c.config.ParseLinks && strings.Contains(irc.Message, "http") {
		go c.linkhandler(irc)
	}

}

// linkhandler replies to messages with http links
func (c *Connection) linkhandler(irc *IRC) {
	if !c.config.ParseLinks {
		return
	}
	defer c.Log.Println("done handling link")
	// word starts with http and is url parsable
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

	c.Log.Println("sending http request:", ss)
	t1 := time.Now()
	resp, err := c.HTTPClient.Do(req)
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
	reader := io.LimitReader(resp.Body, 512)
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		c.Log.Println("error reading from reader:", err)
		// but still reply with response time
		irc.Reply(c, fmt.Sprintf("%s %s (%s)", resp.Status, time.Now().Sub(t1), "read error"))
		return
	}
	meta := getLinkTitleFromHTML(b)
	if meta.Title != "" {
		irc.Reply(c, fmt.Sprintf("%s %s %q (%s)", resp.Status, time.Now().Sub(t1), meta.Title, meta.ContentType))
		return
	}
	irc.Reply(c, fmt.Sprintf("%s %s (%s)", resp.Status, time.Now().Sub(t1), meta.ContentType))

}

type htmlMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"site_name"`
	ContentType string `json:"content_type"`
}

func getLinkTitleFromHTML(htmlbytes []byte) *htmlMeta {
	var reader bytes.Buffer
	reader.Write(htmlbytes)
	z := html.NewTokenizer(&reader)

	titleFound := false

	hm := new(htmlMeta)
	hm.ContentType = http.DetectContentType(htmlbytes)

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
