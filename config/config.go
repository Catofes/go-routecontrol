/*
Package config try to load config from a file and construct a Singleton Config Object.
*/
package config

import (
	"sync"
)

var mainConfig *JSONConfig
var once *sync.Once

// JSONConfig struct save all the config parameters.
type JSONConfig struct {
	Token            string
	UdpListenAddress string
	UdpListenPort    int
	UdpSecretCode    int
}

func (s *JSONConfig) init(path string) *JSONConfig {
	return s
}

// GetInstance return the singleton instance.
func GetInstance(path string) *JSONConfig {
	if path != "" {
		once.Do(func() {
			mainConfig = (&JSONConfig{}).init(path)
		})
	}
	return mainConfig
}
