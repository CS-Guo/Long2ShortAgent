package sequence

import (
	"database/sql"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const sqlReplaceIntoStub = `REPLACE INTO sequence (stub) VALUES ('a')`

type Mysql struct {
	conn sqlx.SqlConn
}

func NewMysql(dsn string) Sequence {
	return &Mysql{
		conn: sqlx.NewMysql(dsn),
	}
}

// Next 取下一个号
func (m *Mysql) Next() (uint64, error) {
	// 预编译
	var stmt sqlx.StmtSession
	stmt, err := m.conn.Prepare(sqlReplaceIntoStub) // 预编译
	if err != nil {
		logx.Errorw("m.conn.Prepare failed", logx.LogField{Key: "err", Value: err.Error()})
		return 0, err
	}
	defer stmt.Close()

	// 执行
	var rest sql.Result
	rest, err = stmt.Exec()
	if err != nil {
		logx.Errorw("stmt.Exec failed", logx.LogField{Key: "err", Value: err.Error()})
		return 0, err
	}

	// 获取插入的ID
	var lid int64
	lid, err = rest.LastInsertId()
	if err != nil {
		logx.Errorw("stmt.Exec failed", logx.LogField{Key: "err", Value: err.Error()})
	}
	return uint64(lid), err
}
