// Package store provides SQLite-backed persistence for the Three-Body Engine.
package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// schemaV1 defines the initial database schema.
const schemaV1 = `
CREATE TABLE IF NOT EXISTS tasks (
	task_id          TEXT PRIMARY KEY,
	current_phase    TEXT NOT NULL DEFAULT 'A',
	status           TEXT NOT NULL DEFAULT 'running',
	state_version    INTEGER NOT NULL DEFAULT 1,
	round            INTEGER NOT NULL DEFAULT 0,
	budget_used_usd  REAL NOT NULL DEFAULT 0.0,
	budget_cap_usd   REAL NOT NULL DEFAULT 0.0,
	last_event_seq   INTEGER NOT NULL DEFAULT 0,
	updated_at_unix  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS workflow_events (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id      TEXT NOT NULL,
	seq_no       INTEGER NOT NULL,
	phase        TEXT NOT NULL,
	event_type   TEXT NOT NULL,
	payload_json TEXT NOT NULL DEFAULT '{}',
	created_at   INTEGER NOT NULL,
	UNIQUE(task_id, seq_no)
);
CREATE INDEX IF NOT EXISTS idx_events_task_seq ON workflow_events(task_id, seq_no);

CREATE TABLE IF NOT EXISTS phase_snapshots (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id       TEXT NOT NULL,
	phase         TEXT NOT NULL,
	round         INTEGER NOT NULL DEFAULT 0,
	snapshot_json TEXT NOT NULL DEFAULT '{}',
	checksum      TEXT NOT NULL DEFAULT '',
	created_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_snapshots_task_phase ON phase_snapshots(task_id, phase);

CREATE TABLE IF NOT EXISTS audit_records (
	id            TEXT PRIMARY KEY,
	task_id       TEXT NOT NULL,
	category      TEXT NOT NULL,
	actor         TEXT NOT NULL DEFAULT '',
	action        TEXT NOT NULL,
	request_json  TEXT NOT NULL DEFAULT '{}',
	decision_json TEXT NOT NULL DEFAULT '{}',
	severity      TEXT NOT NULL DEFAULT 'info',
	created_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_task ON audit_records(task_id);

CREATE TABLE IF NOT EXISTS intent_logs (
	intent_id    TEXT PRIMARY KEY,
	task_id      TEXT NOT NULL,
	worker_id    TEXT NOT NULL DEFAULT '',
	target_file  TEXT NOT NULL,
	operation    TEXT NOT NULL,
	status       TEXT NOT NULL DEFAULT 'pending',
	pre_hash     TEXT NOT NULL DEFAULT '',
	post_hash    TEXT NOT NULL DEFAULT '',
	payload_hash TEXT NOT NULL DEFAULT '',
	lease_until  INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_intents_task_status ON intent_logs(task_id, status);

CREATE TABLE IF NOT EXISTS workers (
	worker_id        TEXT PRIMARY KEY,
	task_id          TEXT NOT NULL,
	phase            TEXT NOT NULL,
	role             TEXT NOT NULL DEFAULT '',
	state            TEXT NOT NULL DEFAULT 'created',
	file_ownership   TEXT NOT NULL DEFAULT '[]',
	soft_timeout_sec INTEGER NOT NULL DEFAULT 300,
	hard_timeout_sec INTEGER NOT NULL DEFAULT 600,
	last_heartbeat   INTEGER NOT NULL DEFAULT 0,
	created_at_unix  INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_workers_task ON workers(task_id, state);

CREATE TABLE IF NOT EXISTS score_cards (
	review_id         TEXT PRIMARY KEY,
	task_id           TEXT NOT NULL,
	reviewer          TEXT NOT NULL,
	correctness       INTEGER NOT NULL DEFAULT 0,
	security          INTEGER NOT NULL DEFAULT 0,
	maintainability   INTEGER NOT NULL DEFAULT 0,
	cost              INTEGER NOT NULL DEFAULT 0,
	delivery_risk     INTEGER NOT NULL DEFAULT 0,
	issues_json       TEXT NOT NULL DEFAULT '[]',
	alternatives_json TEXT NOT NULL DEFAULT '[]',
	verdict           TEXT NOT NULL DEFAULT '',
	created_at        INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_score_cards_task ON score_cards(task_id);

CREATE TABLE IF NOT EXISTS cost_deltas (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id       TEXT NOT NULL,
	input_tokens  INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	amount_usd    REAL NOT NULL DEFAULT 0.0,
	provider      TEXT NOT NULL DEFAULT '',
	phase         TEXT NOT NULL DEFAULT '',
	created_at    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_cost_deltas_task ON cost_deltas(task_id);
`

// NewDB opens a SQLite database at the given path with recommended pragmas
// and runs the V1 schema migration.
func NewDB(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Limit connections to 1 for SQLite (WAL allows concurrent reads but single writer).
	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.ExecContext(context.Background(), schemaV1)
	return err
}
