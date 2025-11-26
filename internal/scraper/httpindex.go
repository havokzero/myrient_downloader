package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"awesomeProject1/internal/domain"

	"golang.org/x/net/html"
)

// HTTPIndex fetches and parses simple HTML directory indexes (like Myrient).
type HTTPIndex struct {
	client *http.Client
}

func NewHTTPIndex() *HTTPIndex {
	return &HTTPIndex{
		client: &http.Client{},
	}
}

// List returns all file/directory entries found at the given URL.
func (h *HTTPIndex) List(rawURL string) ([]domain.FileEntry, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("empty URL")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	resp, err := h.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("http status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var entries []domain.FileEntry

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			// Only consider anchors that live inside <td> (skip header <th> sort links).
			if n.Parent == nil || n.Parent.Type != html.ElementNode || n.Parent.Data != "td" {
				// Not in a data cell, skip (these are usually header arrows etc.).
				// walk children anyway
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				return
			}

			href := ""
			for _, a := range n.Attr {
				if a.Key == "href" {
					href = a.Val
					break
				}
			}
			if href == "" {
				// no target, skip
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				return
			}

			name := strings.TrimSpace(nodeText(n))
			if name == "" {
				// empty label, skip
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				return
			}

			// Resolve relative URLs
			rel, err := url.Parse(href)
			if err != nil {
				// bad href, skip
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				return
			}
			abs := u.ResolveReference(rel)

			isDir := strings.HasSuffix(href, "/") || strings.HasSuffix(name, "/")

			entries = append(entries, domain.FileEntry{
				Name:  name,
				URL:   abs.String(),
				IsDir: isDir,
			})
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return entries, nil
}

// nodeText returns all concatenated text nodes under n.
func nodeText(n *html.Node) string {
	var b strings.Builder
	var rec func(*html.Node)
	rec = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			rec(c)
		}
	}
	rec(n)
	return b.String()
}
