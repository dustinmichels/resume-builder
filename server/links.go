package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

// docLink is a link as written in the resume: the text on the page and where
// it points.
type docLink struct{ Label, URL string }

// docLinks lists the resume's links, without repeats: contacts with a URL and
// [label](url) markup in any field.
func docLinks(doc Doc) []docLink {
	var out []docLink
	add := func(l docLink) {
		if !slices.Contains(out, l) {
			out = append(out, l)
		}
	}
	if doc.Basics != nil {
		for _, c := range doc.Basics.Contacts {
			if strings.TrimSpace(c.Text) != "" && c.URL != "" {
				add(docLink{c.Text, c.URL})
			}
		}
	}
	for _, s := range contentPieces(doc) {
		for _, m := range markupLinkRe.FindAllStringSubmatch(s, -1) {
			add(docLink{m[1], m[2]})
		}
	}
	return out
}

var (
	// urlLikeRe matches link text that reads as an address: a domain with an
	// optional path, or an email address.
	urlLikeRe = regexp.MustCompile(`(?i)^(?:https?://)?[\w-]+(?:\.[\w-]+)*\.[a-z]{2,}(?:[/?#]\S*)?$|^[\w.+-]+@[\w-]+(?:\.[\w-]+)+$`)
	digitRe   = regexp.MustCompile(`\d`)
)

// linkChecks reports links that won't open, whose printed text names a
// different address, or (when facts is set) that aren't clickable in the PDF.
func linkChecks(doc Doc, facts *pdfFacts) []Check {
	links := docLinks(doc)
	if len(links) == 0 {
		return nil
	}
	l := checkList{group: groupLinks}
	var unclickable []string
	for _, link := range links {
		if problem := urlProblem(link.URL); problem != "" {
			l.add("warn", "Link won’t open: "+link.URL, problem)
			continue
		}
		if urlLikeRe.MatchString(link.Label) && linkKey(link.Label) != linkKey(link.URL) {
			l.add("warn", "Link text names a different address: "+link.Label,
				fmt.Sprintf("On paper readers see %s, but the link opens %s.", link.Label, link.URL))
		}
		if facts != nil && !slices.Contains(facts.URLs, link.URL) {
			unclickable = append(unclickable, fmt.Sprintf("%s → %s", link.Label, link.URL))
		}
	}
	if facts != nil {
		if len(unclickable) > 0 {
			l.add("warn", fmt.Sprintf("%d of %d links aren’t clickable in the PDF", len(unclickable), len(links)),
				"The template shows these fields as plain text, so the link markup is printed instead:\n"+sample(unclickable))
		} else {
			l.add("ok", fmt.Sprintf("All %d links are clickable in the PDF", len(links)), "")
		}
	}
	return l.checks
}

// urlProblem says why a PDF viewer couldn't open u, or returns "".
func urlProblem(u string) string {
	p, err := url.Parse(u)
	if err != nil {
		return "Not a valid address."
	}
	switch p.Scheme {
	case "http", "https":
		if p.Host == "" {
			return "No site name after " + p.Scheme + "://."
		}
	case "mailto":
		addr, _, _ := strings.Cut(p.Opaque, "?")
		if _, err := mail.ParseAddress(addr); err != nil {
			return fmt.Sprintf("%q isn’t an email address.", addr)
		}
	case "tel":
		if len(digitRe.FindAllString(p.Opaque, -1)) < 7 {
			return "Too few digits for a phone number."
		}
	case "":
		return "It has no https:// (or mailto:, tel:) in front, so PDF viewers can’t open it."
	default:
		return fmt.Sprintf("Most PDF viewers won’t open %s: links.", p.Scheme)
	}
	return ""
}

// linkKey reduces a link or its printed text to a comparable address:
// lowercase, without scheme, "www." or trailing slash.
func linkKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, prefix := range []string{"mailto:", "https://", "http://", "www."} {
		s = strings.TrimPrefix(s, prefix)
	}
	return strings.TrimSuffix(s, "/")
}

const (
	linkTimeout = 8 * time.Second
	linkTTL     = 10 * time.Minute
)

// linkChecker requests the resume's web links, remembering each answer for
// linkTTL so re-checking while editing doesn't hit the sites again.
type linkChecker struct {
	client *http.Client
	mu     sync.Mutex
	probes map[string]*linkProbe
}

// linkProbe is one request; status and err are set once done is closed.
type linkProbe struct {
	done    chan struct{}
	started time.Time
	status  int
	err     error
}

func newLinkChecker() *linkChecker {
	return &linkChecker{client: &http.Client{Timeout: linkTimeout}, probes: map[string]*linkProbe{}}
}

// probe returns the request for u, starting one unless a recent one exists.
// Requests run detached from callers, so an abandoned check still fills the cache.
func (c *linkChecker) probe(u string) *linkProbe {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.probes[u]; ok && time.Since(p.started) < linkTTL {
		return p
	}
	for k, p := range c.probes {
		if time.Since(p.started) >= linkTTL {
			delete(c.probes, k)
		}
	}
	p := &linkProbe{done: make(chan struct{}), started: time.Now()}
	c.probes[u] = p
	go func() {
		defer close(p.done)
		p.status, p.err = c.fetch(u)
	}()
	return p
}

func (c *linkChecker) fetch(u string) (int, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; resume-builder link check)")
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

// check requests every http(s) link in doc and reports the ones that fail.
func (c *linkChecker) check(ctx context.Context, doc Doc) ([]Check, error) {
	var urls []string
	for _, link := range docLinks(doc) {
		if p, err := url.Parse(link.URL); err == nil && (p.Scheme == "http" || p.Scheme == "https") && p.Host != "" &&
			!slices.Contains(urls, link.URL) {
			urls = append(urls, link.URL)
		}
	}
	probes := make([]*linkProbe, len(urls))
	for i, u := range urls {
		probes[i] = c.probe(u)
	}

	l := checkList{group: groupLinks, checks: []Check{}}
	working := 0
	for i, p := range probes {
		select {
		case <-p.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		switch {
		case p.err != nil:
			l.add("warn", "Broken link: "+urls[i], fetchError(p.err))
		case p.status < 400:
			working++
		case p.status == http.StatusNotFound || p.status == http.StatusGone:
			l.add("warn", "Broken link: "+urls[i], "The site answered "+statusText(p.status)+".")
		default:
			l.add("info", "Couldn’t confirm: "+urls[i],
				"The site answered "+statusText(p.status)+". Some sites (LinkedIn among them) refuse automated checks; open the link to be sure.")
		}
	}
	switch {
	case working > 0 && working == len(urls):
		l.add("ok", fmt.Sprintf("All %d web links open", working), "")
	case working > 0:
		l.add("ok", fmt.Sprintf("%d of %d web links open", working, len(urls)), "")
	}
	return l.checks, nil
}

func statusText(status int) string {
	if text := http.StatusText(status); text != "" {
		return fmt.Sprintf("%d %s", status, text)
	}
	return fmt.Sprint(status)
}

// fetchError describes a failed request without Go's "Get <url>:" prefix.
func fetchError(err error) string {
	if dns, ok := errors.AsType[*net.DNSError](err); ok && dns.IsNotFound {
		return "No such site (" + dns.Name + " doesn’t resolve)."
	}
	if ue, ok := errors.AsType[*url.Error](err); ok {
		if ue.Timeout() {
			return fmt.Sprintf("No answer within %s.", linkTimeout)
		}
		err = ue.Err
	}
	return err.Error() + "."
}
