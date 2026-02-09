package url_tool

import (
	"net/url"
	"path"

	"github.com/zeromicro/go-zero/core/logx"
)

func BasePath(orginUrl string) (string, error) {
	myUrl, err := url.Parse(orginUrl)
	if err != nil {
		logx.Errorw("url.Parse failed", logx.LogField{Key: "error", Value: err})
		return "", err
	}
	return path.Base(myUrl.Path), nil
}
