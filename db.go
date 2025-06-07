package main

import (
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

type DBEntry struct {
	ID              string `gorm:"primaryKey"`
	Author          string
	Title           string
	Status          string
	RulesMatched    string
	RegexGroupsJSON string
	CreatedAt       time.Time
	Updated         string
	Published       string
	PostURL         string
}

/*
initDatabase initializes the SQLite DB and GORM settings.
*/
func initDatabase() {
	var gormLogger logger.Interface
	if debug {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	rulesLock.RLock()
	dbPath := rules.Database.Path
	if dbPath == "" {
		dbPath = "entries.db"
	}
	rulesLock.RUnlock()

	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{Logger: gormLogger})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&DBEntry{})
}

/*
processEntry checks deduplication, stores entries, and prints matches.
*/
func processEntry(entry Entry) {
	debugLog("Processing entry: %s", entry.ID)

	var existing DBEntry
	if err := db.First(&existing, "id = ?", entry.ID).Error; err == nil {
		debugLog("Entry already seen: %s", entry.ID)
		return
	}

	matched, groups := applyRules(entry)

	dbEntry := DBEntry{
		ID:              entry.ID,
		Author:          entry.Author.Name,
		Title:           entry.Title,
		Status:          "unread",
		RulesMatched:    fmt.Sprintf("%v", matched),
		RegexGroupsJSON: fmt.Sprintf("%v", groups),
		CreatedAt:       time.Now(),
		Updated:         entry.Updated,
		Published:       entry.Published,
		PostURL:         entry.Link.Href,
	}

	db.Create(&dbEntry)

	if len(matched) > 0 {
		notifyMatch(entry, matched, groups)
		//fmt.Printf("MATCH: Entry %s matched rules: %v, groups: %v\n", entry.ID, matched, groups)
	} else {
		debugLog("No rules matched for entry: %s", entry.ID)
	}
}
