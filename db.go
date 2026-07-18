package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./wildcard.db")
	if err != nil {
		log.Fatal("DB open error:", err)
	}
	db.SetMaxOpenConns(1)
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA foreign_keys=ON")

	migrate()
	log.Println("✅ Database ready")
}

func migrate() {
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS credentials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		api_token TEXT NOT NULL DEFAULT '',
		zone_id TEXT NOT NULL DEFAULT '',
		domain TEXT NOT NULL DEFAULT '',
		vps_ip TEXT NOT NULL DEFAULT '',
		ssh_host TEXT NOT NULL DEFAULT '',
		ssh_port INTEGER NOT NULL DEFAULT 22,
		ssh_user TEXT NOT NULL DEFAULT 'root',
		ssh_password TEXT NOT NULL DEFAULT '',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`)
	// Add columns if missing
	for _, col := range []string{"ssh_host", "ssh_port", "ssh_user", "ssh_password"} {
		db.Exec("ALTER TABLE credentials ADD COLUMN " + col)
	}
}
