package logic

import (
	"strings"
	"testing"
	"time"
)

func TestExtractContextFromQuery_RelativeRangeAndDomainShortCode(t *testing.T) {
	ctx := extractContextFromQuery("查看sfguo.cn/d近 7天的流量")
	if ctx.ShortUrl != "d" {
		t.Fatalf("expected short code d, got %q", ctx.ShortUrl)
	}
	if ctx.From == "" || ctx.To == "" {
		t.Fatalf("expected non-empty from/to, got from=%q to=%q", ctx.From, ctx.To)
	}
}

func TestExtractContextFromQuery_RelativeExpireAt(t *testing.T) {
	ctx := extractContextFromQuery("修改他的过期时间为7天后")
	if strings.TrimSpace(ctx.ExpireAt) == "" {
		t.Fatalf("expected expireAt extracted")
	}
	if _, err := time.Parse("2006-01-02 15:04:05", ctx.ExpireAt); err != nil {
		t.Fatalf("expected expireAt in yyyy-mm-dd hh:mm:ss format, got %q, err=%v", ctx.ExpireAt, err)
	}
}

func TestParseRelativeRange_ChineseNumber(t *testing.T) {
	from, to := parseRelativeRange("查询近七天流量")
	if from == "" || to == "" {
		t.Fatalf("expected non-empty range, got from=%q to=%q", from, to)
	}
}

func TestParseRelativeExpireAt_Tomorrow(t *testing.T) {
	expireAt := parseRelativeExpireAt("把这个短链过期时间改到明天")
	if expireAt == "" {
		t.Fatalf("expected non-empty expireAt")
	}
	if _, err := time.Parse("2006-01-02 15:04:05", expireAt); err != nil {
		t.Fatalf("invalid expireAt format: %q", expireAt)
	}
}

func TestExtractContextFromQuery_RelativeRangeRecentPeriod(t *testing.T) {
	ctx := extractContextFromQuery("sfguo.cn/d近期 3天的流量")
	if ctx.ShortUrl != "d" {
		t.Fatalf("expected short code d, got %q", ctx.ShortUrl)
	}
	if ctx.From == "" || ctx.To == "" {
		t.Fatalf("expected non-empty from/to, got from=%q to=%q", ctx.From, ctx.To)
	}
}

func TestExtractContextFromQuery_RelativeExpireAtChineseDigits(t *testing.T) {
	ctx := extractContextFromQuery("sfguo.cn/d的过期时间修改为三天后")
	if strings.TrimSpace(ctx.ShortUrl) != "d" {
		t.Fatalf("expected short code d, got %q", ctx.ShortUrl)
	}
	if strings.TrimSpace(ctx.ExpireAt) == "" {
		t.Fatalf("expected expireAt extracted")
	}
	if _, err := time.Parse("2006-01-02 15:04:05", ctx.ExpireAt); err != nil {
		t.Fatalf("expected expireAt in yyyy-mm-dd hh:mm:ss format, got %q, err=%v", ctx.ExpireAt, err)
	}
}
