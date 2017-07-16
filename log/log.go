package log

import (
	"github.com/op/go-logging"
	"sync"
	"os"
)

var instance *logging.Logger

var once sync.Once

func GetInstance() *logging.Logger {
	once.Do(func() {
		instance = logging.MustGetLogger("example")
		backend := logging.NewLogBackend(os.Stdout, "", 0)
		format := logging.MustStringFormatter(
			`%{color}%{time:0102 15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
		)
		backendFormatter := logging.NewBackendFormatter(backend, format)
		backendLeveled := logging.AddModuleLevel(backendFormatter)
		backendLeveled.SetLevel(logging.WARNING, "")
		logging.SetBackend(backendLeveled)

	})
	return instance
}
