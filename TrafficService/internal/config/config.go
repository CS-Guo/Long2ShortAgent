package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf

	ShortUrlDB struct {
		DSN string
	}

	Redis struct {
		Host string
		Type string
		Pass string
	}

	Event struct {
		Stream    string
		Group     string
		Consumer  string
		BatchSize int
		BlockMs   int
	}
}
