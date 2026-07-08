package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

const profileDBName = "profile.db"

// store 仅管理 profile.db（profiles 表）。历史发言数据由 statistics 插件通过
// statistics.query_messages 能力提供，本插件不直接读 statistics.db。
type store struct {
	profile *sql.DB
	mu      sync.Mutex
}

func openStore() (*store, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取插件路径失败: %w", err)
	}
	dbPath := filepath.Join(filepath.Dir(exe), profileDBName)

	profile, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开画像数据库失败: %w", err)
	}
	if err := profile.Ping(); err != nil {
		_ = profile.Close()
		return nil, fmt.Errorf("连接画像数据库失败: %w", err)
	}

	if _, err := profile.Exec(`CREATE TABLE IF NOT EXISTS profiles (
		chatroom  TEXT NOT NULL DEFAULT '',
		member    TEXT NOT NULL,
		profile   TEXT,
		last_msg_id INTEGER DEFAULT 0,
		updated_at DATETIME,
		PRIMARY KEY (chatroom, member)
	);`); err != nil {
		_ = profile.Close()
		return nil, fmt.Errorf("创建画像表失败: %w", err)
	}

	return &store{profile: profile}, nil
}

func (s *store) Close() error {
	if s == nil || s.profile == nil {
		return nil
	}
	return s.profile.Close()
}

// loadProfile 读取已有画像；返回 (record, found)
func (s *store) loadProfile(chatroom, member string) (profileRecord, bool) {
	var rec profileRecord
	err := s.profile.QueryRow(
		`SELECT chatroom, member, profile, last_msg_id, updated_at FROM profiles WHERE chatroom=? AND member=?`,
		chatroom, member,
	).Scan(&rec.Chatroom, &rec.Member, &rec.Profile, &rec.LastMsgID, &rec.UpdatedAt)
	if err != nil {
		return profileRecord{}, false
	}
	return rec, true
}

// saveProfile 新增或更新画像（覆盖写）。updated_at 用本地时间，与 statistics 表 timestamp 一致。
func (s *store) saveProfile(rec profileRecord) error {
	_, err := s.profile.Exec(
		`INSERT INTO profiles (chatroom, member, profile, last_msg_id, updated_at)
		VALUES (?, ?, ?, ?, datetime('now','localtime'))
		ON CONFLICT(chatroom, member) DO UPDATE SET
			profile=excluded.profile,
			last_msg_id=excluded.last_msg_id,
			updated_at=datetime('now','localtime');`,
		rec.Chatroom, rec.Member, rec.Profile, rec.LastMsgID,
	)
	return err
}

var errStoreNotReady = errors.New("画像存储未初始化")
