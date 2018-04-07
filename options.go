package diskqueue

import (
	"crypto/md5"
	"hash/crc32"
	"io"
	"log"
	"os"
)

type Options struct {
	ID         int64  `yaml:"node-id"`
	TCPAddress string `yaml:"tcp-address"`
	DataPath   string `yaml:"data-path"`
}

func NewOptions() *Options {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	h := md5.New()
	io.WriteString(h, hostname)
	defaultID := int64(crc32.ChecksumIEEE(h.Sum(nil)) % 1024)

	return &Options{
		ID: defaultID,
	}
}
