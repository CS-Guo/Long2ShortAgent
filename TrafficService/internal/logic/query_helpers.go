package logic

import (
	"context"

	"trafficservice/internal/svc"
	"trafficservice/internal/types"
)

type pvuvRow struct {
	PV int64 `db:"pv"`
	UV int64 `db:"uv"`
}

type trendRow struct {
	Date string `db:"stat_date"`
	PV   int64  `db:"pv"`
	UV   int64  `db:"uv"`
}

type referrerRow struct {
	Referrer string `db:"referer"`
	PV       int64  `db:"pv"`
}

// queryPVUV 查询时间区间内的 PV/UV 汇总。
func queryPVUV(ctx context.Context, svcCtx *svc.ServiceContext, shortCode, from, to string) (*pvuvRow, error) {
	query := `
select
  ifnull(sum(pv), 0) as pv,
  ifnull(sum(uv), 0) as uv
from short_url_daily_stat
where short_code = ?
  and stat_date >= ?
  and stat_date <= ?`

	var row pvuvRow
	if err := svcCtx.DB.QueryRowCtx(ctx, &row, query, shortCode, from, to); err != nil {
		return nil, err
	}
	return &row, nil
}

// queryTrend 查询时间区间内的日趋势数据。
func queryTrend(ctx context.Context, svcCtx *svc.ServiceContext, shortCode, from, to string) ([]types.DailyStat, error) {
	query := `
select
  cast(stat_date as char) as stat_date,
  pv,
  uv
from short_url_daily_stat
where short_code = ?
  and stat_date >= ?
  and stat_date <= ?
order by stat_date asc`

	var rows []trendRow
	if err := svcCtx.DB.QueryRowsCtx(ctx, &rows, query, shortCode, from, to); err != nil {
		return nil, err
	}

	out := make([]types.DailyStat, 0, len(rows))
	for _, item := range rows {
		out = append(out, types.DailyStat{
			Date: item.Date,
			Pv:   item.PV,
			Uv:   item.UV,
		})
	}

	return out, nil
}

// queryTopReferrers 查询来源站点 Top5。
func queryTopReferrers(ctx context.Context, svcCtx *svc.ServiceContext, shortCode, from, to string) ([]types.ReferrerStat, error) {
	query := `
select
  referer,
  count(1) as pv
from short_url_click_event
where short_code = ?
  and date(occurred_at) >= ?
  and date(occurred_at) <= ?
group by referer
order by pv desc
limit 5`

	var rows []referrerRow
	if err := svcCtx.DB.QueryRowsCtx(ctx, &rows, query, shortCode, from, to); err != nil {
		return nil, err
	}

	out := make([]types.ReferrerStat, 0, len(rows))
	for _, item := range rows {
		out = append(out, types.ReferrerStat{
			Referrer: item.Referrer,
			Pv:       item.PV,
		})
	}

	return out, nil
}
