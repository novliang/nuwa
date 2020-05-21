package nuwa

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type Nuwa struct {
	Logger echo.Logger
}

// 女娲开始用石头补天了
func (n *Nuwa) Work(s Stone, args ...interface{}) {
	n.Logger.Fatal(s.Work(args))
}

// 天漏了，创建一个女娲准备补天了
func New(n *Nuwa) {
	n = &Nuwa{
		Logger: log.New("nuwa"),
	}
	return
}
