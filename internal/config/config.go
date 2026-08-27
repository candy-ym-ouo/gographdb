// Package config 负责命令行配置。
package config

import (
	"flag"
	"time"
)

// Config 是服务运行参数。
type Config struct {
	Addr             string
	DataDir          string
	SnapshotInterval time.Duration
}

// Parse 解析命令行参数。
func Parse() Config {
	var c Config
	flag.StringVar(&c.Addr, "addr", "127.0.0.1:8080", "HTTP listen address")
	flag.StringVar(&c.DataDir, "data", "./data", "data directory")
	flag.DurationVar(&c.SnapshotInterval, "snapshot-interval", 30*time.Second, "automatic snapshot interval (0 disables)")
	flag.Parse()
	return c
}
