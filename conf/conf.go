package conf

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type Conf struct {
	LogPath  string `yaml:"logpath"`
	LogLevel int    `yaml:"loglevel"`
	DataPath string `yaml:"datapath"`
}

func NewConf(path string, conf *Conf) error {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(file, conf)
}
