package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sbgayhub/golem/sdk/message"

	_ "modernc.org/sqlite"
)

const (
	analysisDBName   = "analysis.db"
	statisticsDBName = "statistics.db"
	defaultRankLimit = 10
)

var errInvalidMessage = errors.New("invalid message")

type store struct {
	analysis   *sql.DB
	statistics *sql.DB
}

type rankEntry struct {
	Member string
	Count  int
	Detail string
}

type typeCount struct {
	Type  string
	Count int
}

type totalSummary struct {
	Speakers int
	Messages int
	Types    []typeCount
}

func openStore(dir string) (*store, error) {
	if dir == "" {
		executable, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("获取插件路径失败: %w", err)
		}
		dir = filepath.Dir(executable)
	}

	analysis, err := sql.Open("sqlite", filepath.Join(dir, analysisDBName))
	if err != nil {
		return nil, fmt.Errorf("打开分析数据库失败: %w", err)
	}

	statistics, err := sql.Open("sqlite", filepath.Join(dir, statisticsDBName))
	if err != nil {
		_ = analysis.Close()
		return nil, fmt.Errorf("打开统计数据库失败: %w", err)
	}

	st := &store{analysis: analysis, statistics: statistics}
	if err := st.Init(); err != nil {
		_ = st.Close()
		return nil, err
	}
	return st, nil
}

func (s *store) Init() error {
	if err := s.Ping(); err != nil {
		return err
	}

	if _, err := s.analysis.Exec(`CREATE TABLE IF NOT EXISTS analysis (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		sender TEXT NOT NULL,
		member TEXT NOT NULL,
		type TEXT NOT NULL,
		count INTEGER DEFAULT 1 NOT NULL,
		date TEXT NOT NULL DEFAULT current_date,
		UNIQUE(sender, member, type, date)
	);`); err != nil {
		return fmt.Errorf("创建分析表失败: %w", err)
	}

	if _, err := s.statistics.Exec(`CREATE TABLE IF NOT EXISTS statistics (
		id INTEGER PRIMARY KEY,
		type TEXT NOT NULL,
		content TEXT,
		sender TEXT,
		receiver TEXT,
		member TEXT,
		raw TEXT NOT NULL,
		timestamp DATETIME DEFAULT current_timestamp
	);`); err != nil {
		return fmt.Errorf("创建统计表失败: %w", err)
	}

	if _, err := s.statistics.Exec(`CREATE INDEX IF NOT EXISTS idx_statistics_sender ON statistics(sender);`); err != nil {
		return fmt.Errorf("创建统计索引失败: %w", err)
	}
	return nil
}

func (s *store) Ping() error {
	if s == nil || s.analysis == nil || s.statistics == nil {
		return errors.New("store is not initialized")
	}
	if err := s.analysis.Ping(); err != nil {
		return fmt.Errorf("分析数据库连接失败: %w", err)
	}
	if err := s.statistics.Ping(); err != nil {
		return fmt.Errorf("统计数据库连接失败: %w", err)
	}
	return nil
}

func (s *store) Close() error {
	var result error
	if s.analysis != nil {
		result = errors.Join(result, s.analysis.Close())
	}
	if s.statistics != nil {
		result = errors.Join(result, s.statistics.Close())
	}
	return result
}

func (s *store) record(msg *message.Message) (bool, error) {
	if msg.GetId() == 0 {
		return false, errInvalidMessage
	}

	sender := msg.GetSender().GetUsername()
	member := msg.GetMember().GetUsername()
	timestamp := msg.GetTimestamp()
	typ := message.TypeUnknown.GetDesc()
	if msg.GetType() != nil && msg.GetType().GetDesc() != "" {
		typ = msg.GetType().GetDesc()
	}

	result, err := s.statistics.Exec(
		`INSERT OR IGNORE INTO statistics (id, type, content, sender, receiver, member, raw, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime(?, 'unixepoch', 'localtime'));`,
		msg.GetId(), typ, msg.GetContent(), sender, msg.GetReceiver().GetUsername(), member, msg.GetRaw(), timestamp)
	if err != nil {
		return false, fmt.Errorf("新增统计数据失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("读取统计写入结果失败: %w", err)
	}
	if affected == 0 {
		return false, nil
	}

	if _, err := s.analysis.Exec(
		`INSERT INTO analysis (sender, member, type, date)
		VALUES (?, ?, ?, date(datetime(?, 'unixepoch'), 'localtime'))
		ON CONFLICT(sender, member, type, date) DO UPDATE SET count = count + 1;`,
		sender, member, typ, timestamp); err != nil {
		return false, fmt.Errorf("新增分析数据失败: %w", err)
	}
	return true, nil
}

func (s *store) QueryRank(sender, dateFilter string, limit int) ([]rankEntry, error) {
	if limit <= 0 {
		limit = defaultRankLimit
	}

	query := `SELECT member, SUM(count) AS count FROM analysis WHERE sender = ? ` + dateFilter + ` GROUP BY member ORDER BY count DESC LIMIT ?;`
	rows, err := s.analysis.Query(query, sender, limit)
	if err != nil {
		return nil, fmt.Errorf("查询排行失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []rankEntry
	for rows.Next() {
		var entry rankEntry
		if err := rows.Scan(&entry.Member, &entry.Count); err != nil {
			return nil, fmt.Errorf("绑定排行数据失败: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历排行数据失败: %w", err)
	}
	return entries, nil
}

func (s *store) QueryMemberTypeCounts(sender, member, dateFilter string) ([]typeCount, error) {
	query := `SELECT type, SUM(count) AS count FROM analysis WHERE sender = ? AND member = ? ` + dateFilter + ` GROUP BY type ORDER BY count DESC;`
	rows, err := s.analysis.Query(query, sender, member)
	if err != nil {
		return nil, fmt.Errorf("查询成员发言详情失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var counts []typeCount
	for rows.Next() {
		var item typeCount
		if err := rows.Scan(&item.Type, &item.Count); err != nil {
			return nil, fmt.Errorf("绑定成员发言详情失败: %w", err)
		}
		counts = append(counts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历成员发言详情失败: %w", err)
	}
	return counts, nil
}

func (s *store) QueryTotal(sender, dateFilter string) (totalSummary, error) {
	var total totalSummary
	var messages sql.NullInt64
	query := `SELECT COUNT(DISTINCT member), SUM(count) FROM analysis WHERE sender = ? ` + dateFilter + `;`
	if err := s.analysis.QueryRow(query, sender).Scan(&total.Speakers, &messages); err != nil {
		return total, fmt.Errorf("查询统计汇总失败: %w", err)
	}
	if messages.Valid {
		total.Messages = int(messages.Int64)
	}

	query = `SELECT type, SUM(count) AS count FROM analysis WHERE sender = ? ` + dateFilter + ` GROUP BY type ORDER BY count DESC;`
	rows, err := s.analysis.Query(query, sender)
	if err != nil {
		return total, fmt.Errorf("查询消息类型汇总失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var item typeCount
		if err := rows.Scan(&item.Type, &item.Count); err != nil {
			return total, fmt.Errorf("绑定消息类型汇总失败: %w", err)
		}
		total.Types = append(total.Types, item)
	}
	if err := rows.Err(); err != nil {
		return total, fmt.Errorf("遍历消息类型汇总失败: %w", err)
	}
	return total, nil
}
