package types

type TrafficSummaryRequest struct {
	ShortCode string `path:"shortCode"`
	From      string `form:"from"`
	To        string `form:"to"`
}

type ReferrerStat struct {
	Referrer string `json:"referrer"`
	Pv       int64  `json:"pv"`
}

type DailyStat struct {
	Date string `json:"date"`
	Pv   int64  `json:"pv"`
	Uv   int64  `json:"uv"`
}

type TrafficSummaryResponse struct {
	ShortCode    string         `json:"shortCode"`
	From         string         `json:"from"`
	To           string         `json:"to"`
	Pv           int64          `json:"pv"`
	Uv           int64          `json:"uv"`
	TopReferrers []ReferrerStat `json:"topReferrers"`
	Trend        []DailyStat    `json:"trend"`
}
