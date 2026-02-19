package securitylog

import (
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/navantesolutions/apimcore/internal/hub"
)

const asyncBuffer = 2000

type Logger interface {
	Append(ev hub.TrafficEvent)
	Close() error
}

func New(path string) (Logger, error) {
	if path == "" {
		return nil, nil
	}
	if strings.HasPrefix(path, "sqlite:") {
		dbPath := strings.TrimPrefix(path, "sqlite:")
		return newSQLite(dbPath)
	}
	return newFile(path)
}

type fileLogger struct {
	ch   chan hub.TrafficEvent
	done chan struct{}
	once sync.Once
}

func newFile(path string) (Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	l := &fileLogger{ch: make(chan hub.TrafficEvent, asyncBuffer), done: make(chan struct{})}
	go l.runFile(f)
	return l, nil
}

func (l *fileLogger) runFile(f *os.File) {
	defer f.Close()
	for {
		select {
		case ev := <-l.ch:
			line := eventLine(ev)
			if len(line) > 0 {
				_, _ = f.Write(line)
				_, _ = f.Write([]byte("\n"))
			}
		case <-l.done:
			for {
				select {
				case ev := <-l.ch:
					line := eventLine(ev)
					if len(line) > 0 {
						_, _ = f.Write(line)
						_, _ = f.Write([]byte("\n"))
					}
				default:
					return
				}
			}
		}
	}
}

func (l *fileLogger) Close() error {
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		close(l.done)
	})
	return nil
}

func (l *fileLogger) Append(ev hub.TrafficEvent) {
	if l == nil || l.ch == nil {
		return
	}
	if ev.Action != "BLOCKED" && ev.Action != "RATE_LIMIT" {
		return
	}
	select {
	case l.ch <- ev:
	default:
	}
}

const createTableSQL = `CREATE TABLE IF NOT EXISTS security_events (
	time TEXT,
	action TEXT,
	ip TEXT,
	country TEXT,
	method TEXT,
	path TEXT,
	status INTEGER,
	tenant_id TEXT
);`

type sqliteLogger struct {
	ch   chan hub.TrafficEvent
	done chan struct{}
	once sync.Once
	db   *sql.DB
}

func newSQLite(dbPath string) (Logger, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(createTableSQL); err != nil {
		_ = db.Close()
		return nil, err
	}
	l := &sqliteLogger{
		ch:   make(chan hub.TrafficEvent, asyncBuffer),
		done: make(chan struct{}),
		db:   db,
	}
	go l.runSQLite()
	return l, nil
}

func (l *sqliteLogger) runSQLite() {
	insert, err := l.db.Prepare(`INSERT INTO security_events (time, action, ip, country, method, path, status, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return
	}
	defer insert.Close()
	for {
		select {
		case ev := <-l.ch:
			if ev.Action != "BLOCKED" && ev.Action != "RATE_LIMIT" {
				continue
			}
			_, _ = insert.Exec(
				ev.Timestamp.Format(time.RFC3339),
				ev.Action,
				ev.IP,
				ev.Country,
				ev.Method,
				ev.Path,
				ev.Status,
				ev.TenantID,
			)
		case <-l.done:
			for {
				select {
				case ev := <-l.ch:
					if ev.Action != "BLOCKED" && ev.Action != "RATE_LIMIT" {
						continue
					}
					_, _ = insert.Exec(
						ev.Timestamp.Format(time.RFC3339),
						ev.Action,
						ev.IP,
						ev.Country,
						ev.Method,
						ev.Path,
						ev.Status,
						ev.TenantID,
					)
				default:
					_ = l.db.Close()
					return
				}
			}
		}
	}
}

func (l *sqliteLogger) Close() error {
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		close(l.done)
	})
	return nil
}

func (l *sqliteLogger) Append(ev hub.TrafficEvent) {
	if l == nil || l.ch == nil {
		return
	}
	if ev.Action != "BLOCKED" && ev.Action != "RATE_LIMIT" {
		return
	}
	select {
	case l.ch <- ev:
	default:
	}
}

func eventLine(ev hub.TrafficEvent) []byte {
	if ev.Action != "BLOCKED" && ev.Action != "RATE_LIMIT" {
		return nil
	}
	return eventLineAny(ev)
}

func eventLineAny(ev hub.TrafficEvent) []byte {
	row := struct {
		Time     string `json:"time"`
		Action   string `json:"action"`
		IP       string `json:"ip"`
		Country  string `json:"country"`
		Method   string `json:"method"`
		Path     string `json:"path"`
		Status   int    `json:"status"`
		TenantID string `json:"tenant_id,omitempty"`
		Latency  int64  `json:"latency_ms,omitempty"`
		Backend  string `json:"backend,omitempty"`
	}{
		Time:     ev.Timestamp.Format(time.RFC3339),
		Action:   ev.Action,
		IP:       ev.IP,
		Country:  ev.Country,
		Method:   ev.Method,
		Path:     ev.Path,
		Status:   ev.Status,
		TenantID: ev.TenantID,
		Latency:  ev.Latency,
		Backend:  ev.Backend,
	}
	b, _ := json.Marshal(row)
	return b
}

type allTrafficFileLogger struct {
	ch   chan hub.TrafficEvent
	done chan struct{}
	once sync.Once
}

func NewFileLoggerAll(path string) (Logger, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	l := &allTrafficFileLogger{ch: make(chan hub.TrafficEvent, asyncBuffer), done: make(chan struct{})}
	go l.run(f)
	return l, nil
}

func (l *allTrafficFileLogger) run(f *os.File) {
	defer f.Close()
	for {
		select {
		case ev := <-l.ch:
			line := eventLineAny(ev)
			if len(line) > 0 {
				_, _ = f.Write(line)
				_, _ = f.Write([]byte("\n"))
			}
		case <-l.done:
			for {
				select {
				case ev := <-l.ch:
					line := eventLineAny(ev)
					if len(line) > 0 {
						_, _ = f.Write(line)
						_, _ = f.Write([]byte("\n"))
					}
				default:
					return
				}
			}
		}
	}
}

func (l *allTrafficFileLogger) Close() error {
	if l == nil || l.ch == nil {
		return nil
	}
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *allTrafficFileLogger) Append(ev hub.TrafficEvent) {
	if l == nil || l.ch == nil {
		return
	}
	select {
	case l.ch <- ev:
	default:
	}
}
