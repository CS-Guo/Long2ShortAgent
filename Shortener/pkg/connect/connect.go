package connect

import (
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

var client = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
	},
	Timeout: time.Second * 10,
}

func Get(url string) bool {

	resp, err := client.Get(url)
	if err != nil {
		logx.Errorw("client.Get failed ", logx.LogField{Key: "err", Value: err.Error()})
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
