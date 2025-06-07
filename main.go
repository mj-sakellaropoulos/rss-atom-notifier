package main

import (
	"flag"
	"log"
	"time"
)

/*
main initializes the database, loads config, and begins the polling loop.
*/
func main() {
	configPath := flag.String("config", "rules.yaml", "Path to YAML config file")
	flag.Parse()

	loadRules(*configPath)
	initDatabase()
	watchConfig(*configPath)

	for {
		rulesLock.RLock()
		interval := time.Duration(rules.HTTP.PollingIntervalMs) * time.Millisecond
		userAgent := rules.HTTP.UserAgent
		rssURL := rules.HTTP.RSSURL
		rulesLock.RUnlock()

		entries, err := fetchFeed(rssURL, userAgent)
		if err != nil {
			log.Println("Error fetching feed:", err)
			time.Sleep(interval)
			continue
		}
		for _, entry := range entries {
			processEntry(entry)
		}
		time.Sleep(interval)
	}
}
