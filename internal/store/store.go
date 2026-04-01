package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ conn *sql.DB }

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	conn, err := sql.Open("sqlite", filepath.Join(dataDir, "rangeland.db"))
	if err != nil {
		return nil, err
	}
	conn.Exec("PRAGMA journal_mode=WAL")
	conn.Exec("PRAGMA busy_timeout=5000")
	conn.SetMaxOpenConns(4)
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) Conn() *sql.DB { return db.conn }
func (db *DB) Close() error  { return db.conn.Close() }

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
CREATE TABLE IF NOT EXISTS sites (
    id TEXT PRIMARY KEY,
    domain TEXT NOT NULL UNIQUE,
    name TEXT DEFAULT '',
    public INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS pageviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id TEXT NOT NULL,
    path TEXT DEFAULT '/',
    referrer TEXT DEFAULT '',
    referrer_domain TEXT DEFAULT '',
    country TEXT DEFAULT '',
    device TEXT DEFAULT 'desktop',
    browser TEXT DEFAULT '',
    os TEXT DEFAULT '',
    screen TEXT DEFAULT '',
    visitor_hash TEXT DEFAULT '',
    session_hash TEXT DEFAULT '',
    timestamp TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_pv_site ON pageviews(site_id);
CREATE INDEX IF NOT EXISTS idx_pv_time ON pageviews(timestamp);
CREATE INDEX IF NOT EXISTS idx_pv_site_time ON pageviews(site_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_pv_visitor ON pageviews(visitor_hash);
`)
	return err
}

// --- Site ---

type Site struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	Name      string `json:"name"`
	Public    bool   `json:"public"`
	CreatedAt string `json:"created_at"`
}

func (db *DB) CreateSite(domain, name string) (*Site, error) {
	id := "site_" + genID(6)
	now := time.Now().UTC().Format(time.RFC3339)
	if name == "" {
		name = domain
	}
	_, err := db.conn.Exec("INSERT INTO sites (id,domain,name,created_at) VALUES (?,?,?,?)", id, domain, name, now)
	if err != nil {
		return nil, err
	}
	return &Site{ID: id, Domain: domain, Name: name, CreatedAt: now}, nil
}

func (db *DB) ListSites() ([]Site, error) {
	rows, err := db.conn.Query("SELECT id,domain,name,public,created_at FROM sites ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Site
	for rows.Next() {
		var s Site
		var pub int
		rows.Scan(&s.ID, &s.Domain, &s.Name, &pub, &s.CreatedAt)
		s.Public = pub == 1
		out = append(out, s)
	}
	return out, rows.Err()
}

func (db *DB) GetSite(id string) (*Site, error) {
	var s Site
	var pub int
	err := db.conn.QueryRow("SELECT id,domain,name,public,created_at FROM sites WHERE id=?", id).
		Scan(&s.ID, &s.Domain, &s.Name, &pub, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	s.Public = pub == 1
	return &s, nil
}

func (db *DB) GetSiteByDomain(domain string) (*Site, error) {
	var s Site
	var pub int
	err := db.conn.QueryRow("SELECT id,domain,name,public,created_at FROM sites WHERE domain=?", domain).
		Scan(&s.ID, &s.Domain, &s.Name, &pub, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	s.Public = pub == 1
	return &s, nil
}

func (db *DB) DeleteSite(id string) error {
	db.conn.Exec("DELETE FROM pageviews WHERE site_id=?", id)
	_, err := db.conn.Exec("DELETE FROM sites WHERE id=?", id)
	return err
}

// --- Pageview recording ---

// HashVisitor creates a daily-rotating privacy-preserving visitor hash.
// No cookies, no persistent IDs. Same visitor on same day = same hash.
func HashVisitor(siteID, ip, ua string) string {
	day := time.Now().UTC().Format("2006-01-02")
	h := sha256.Sum256([]byte(siteID + ip + ua + day))
	return hex.EncodeToString(h[:8])
}

func HashSession(visitorHash, path string) string {
	h := sha256.Sum256([]byte(visitorHash + time.Now().UTC().Truncate(30*time.Minute).String()))
	return hex.EncodeToString(h[:8])
}

func (db *DB) RecordPageview(siteID, path, referrer, refDomain, country, device, browser, os, screen, visitorHash, sessionHash string) error {
	_, err := db.conn.Exec(`INSERT INTO pageviews (site_id,path,referrer,referrer_domain,country,device,browser,os,screen,visitor_hash,session_hash)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`, siteID, path, referrer, refDomain, country, device, browser, os, screen, visitorHash, sessionHash)
	return err
}

// --- Analytics queries ---

type TopEntry struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type DailyStat struct {
	Date     string `json:"date"`
	Views    int    `json:"views"`
	Visitors int    `json:"visitors"`
}

func (db *DB) Overview(siteID string, days int) map[string]any {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")

	var views, visitors, sessions int
	db.conn.QueryRow("SELECT COUNT(*) FROM pageviews WHERE site_id=? AND timestamp>=?", siteID, cutoff).Scan(&views)
	db.conn.QueryRow("SELECT COUNT(DISTINCT visitor_hash) FROM pageviews WHERE site_id=? AND timestamp>=?", siteID, cutoff).Scan(&visitors)
	db.conn.QueryRow("SELECT COUNT(DISTINCT session_hash) FROM pageviews WHERE site_id=? AND timestamp>=?", siteID, cutoff).Scan(&sessions)

	viewsPerVisitor := 0.0
	if visitors > 0 {
		viewsPerVisitor = float64(views) / float64(visitors)
	}

	// Bounce rate: sessions with only 1 pageview
	var bounceSessions int
	db.conn.QueryRow(`SELECT COUNT(*) FROM (
		SELECT session_hash, COUNT(*) as c FROM pageviews WHERE site_id=? AND timestamp>=? GROUP BY session_hash HAVING c=1
	)`, siteID, cutoff).Scan(&bounceSessions)
	bounceRate := 0.0
	if sessions > 0 {
		bounceRate = float64(bounceSessions) / float64(sessions) * 100
	}

	return map[string]any{
		"pageviews": views, "visitors": visitors, "sessions": sessions,
		"views_per_visitor": fmt.Sprintf("%.1f", viewsPerVisitor),
		"bounce_rate":       fmt.Sprintf("%.1f", bounceRate),
	}
}

func (db *DB) DailyStats(siteID string, days int) []DailyStat {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	rows, err := db.conn.Query(`SELECT date(timestamp) as d, COUNT(*), COUNT(DISTINCT visitor_hash)
		FROM pageviews WHERE site_id=? AND date(timestamp)>=? GROUP BY d ORDER BY d`, siteID, cutoff)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []DailyStat
	for rows.Next() {
		var s DailyStat
		rows.Scan(&s.Date, &s.Views, &s.Visitors)
		out = append(out, s)
	}
	return out
}

func (db *DB) TopPages(siteID string, days, limit int) []TopEntry {
	return db.topQuery("path", siteID, days, limit)
}

func (db *DB) TopReferrers(siteID string, days, limit int) []TopEntry {
	return db.topQuery("referrer_domain", siteID, days, limit)
}

func (db *DB) TopCountries(siteID string, days, limit int) []TopEntry {
	return db.topQuery("country", siteID, days, limit)
}

func (db *DB) TopBrowsers(siteID string, days, limit int) []TopEntry {
	return db.topQuery("browser", siteID, days, limit)
}

func (db *DB) TopOS(siteID string, days, limit int) []TopEntry {
	return db.topQuery("os", siteID, days, limit)
}

func (db *DB) TopDevices(siteID string, days, limit int) []TopEntry {
	return db.topQuery("device", siteID, days, limit)
}

func (db *DB) topQuery(col, siteID string, days, limit int) []TopEntry {
	if limit <= 0 {
		limit = 10
	}
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	// col is a known column name set internally, not user input
	rows, err := db.conn.Query(
		fmt.Sprintf("SELECT %s, COUNT(*) as c FROM pageviews WHERE site_id=? AND timestamp>=? AND %s!='' GROUP BY %s ORDER BY c DESC LIMIT ?", col, col, col),
		siteID, cutoff, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []TopEntry
	for rows.Next() {
		var e TopEntry
		rows.Scan(&e.Name, &e.Count)
		out = append(out, e)
	}
	return out
}

func (db *DB) RealtimeVisitors(siteID string) int {
	cutoff := time.Now().Add(-5 * time.Minute).UTC().Format("2006-01-02 15:04:05")
	var count int
	db.conn.QueryRow("SELECT COUNT(DISTINCT visitor_hash) FROM pageviews WHERE site_id=? AND timestamp>=?", siteID, cutoff).Scan(&count)
	return count
}

// --- Stats ---

func (db *DB) Stats() map[string]any {
	var sites, pageviews int
	db.conn.QueryRow("SELECT COUNT(*) FROM sites").Scan(&sites)
	db.conn.QueryRow("SELECT COUNT(*) FROM pageviews").Scan(&pageviews)
	return map[string]any{"sites": sites, "pageviews": pageviews}
}

func (db *DB) Cleanup(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	res, err := db.conn.Exec("DELETE FROM pageviews WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DB) MonthlyPageviews(siteID string) (int, error) {
	cutoff := time.Now().AddDate(0, -1, 0).Format("2006-01-02 15:04:05")
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM pageviews WHERE site_id=? AND timestamp>=?", siteID, cutoff).Scan(&count)
	return count, err
}

func genID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
