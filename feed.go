package main

import (
	"github.com/mmcdole/gofeed"
	"net/http"
	"time"
)

/*
fetchFeed retrieves the RSS/Atom feed and converts entries to internal Entry structs.
*/
func fetchFeed(url, userAgent string) ([]Entry, error) {
	debugLog("Fetching feed from %s", url)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	fp := gofeed.NewParser()
	fp.Client = client
	fp.UserAgent = userAgent

	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, item := range feed.Items {
		entry := Entry{
			ID:        item.GUID,
			Author:    Author{Name: getAuthor(item)},
			Title:     item.Title,
			Link:      Link{Href: item.Link},
			Published: item.Published,
			Updated:   item.Updated,
			Raw:       item.Content,
		}
		if entry.ID == "" {
			entry.ID = item.Link // fallback if GUID is missing
		}
		entries = append(entries, entry)
	}

	debugLog("Fetched %d entries", len(entries))
	return entries, nil
}

func getAuthor(item *gofeed.Item) string {
	if item.Author != nil {
		return item.Author.Name
	}
	return ""
}

/*
Entry and related types used internally
*/
type Entry struct {
	ID        string
	Author    Author
	Title     string
	Link      Link
	Raw       string
	Published string
	Updated   string
}

type Author struct {
	Name string
}

type Link struct {
	Href string
}
