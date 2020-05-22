package gold

import (
	"github.com/facebookgo/grace/gracehttp"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/novliang/nuwa/utils"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	HeaderOauthServer = "NUWA-OAUTH-SERVER"
)

// 金子是一个Web服务框架
type Gold struct {
	*echo.Echo
	Configurator *viper.Viper
}

// 金子的默认返回格式
type Response struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// 金子配置文件
type GoldConfig struct {
	HttpErrorHandler func(error, echo.Context)
	Routers          func(*Gold)
	OauthClients     []*OauthClient
}

// 加载配置文件
func (g *Gold) LoadConfig() {
	w, err := os.Getwd()
	if err != nil {
		panic("Can't get path wd err")
	}
	a, err := filepath.Abs(filepath.Dir(os.Args[0]))

	if err != nil {
		panic("Can't get path wd err")
	}

	var configFile = "api.toml"
	appConfigPath := filepath.Join(w, "config", configFile)
	if !utils.FileExists(appConfigPath) {
		appConfigPath = filepath.Join(a, "config", configFile)
		if !utils.FileExists(appConfigPath) {
			panic("Can't get db config file err")
		}
	}

	// 配置文件
	g.Configurator = viper.New()

	g.Configurator.SetConfigName("api")
	g.Configurator.AddConfigPath(strings.TrimRight(appConfigPath, configFile))
	err = g.Configurator.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic("Can't get api config file err")
		} else {
			panic(err)
		}
	}
}

// 创建一个新的金子
func NewGoldWithConfig(config GoldConfig) *Gold {

	// original web framework is echo
	e := echo.New()

	// transfer into gold
	g := &Gold{
		Echo: e,
	}

	// load config file
	g.LoadConfig()

	// http error handler
	if config.HttpErrorHandler != nil {
		g.HTTPErrorHandler = config.HttpErrorHandler
	} else {
		g.HTTPErrorHandler = DefaultHttpErrorHandler
	}

	// start logger
	g.Use(middleware.Logger())

	// start recover
	g.Use(middleware.Recover())

	// set CORS
	g.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, HeaderOauthServer},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
	}))

	// Todo support custom validator
	//g.Validator = &Validator

	//Extending
	g.Use(func(handlerFunc echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			think := &Context{c}
			return handlerFunc(think)
		}
	})

	// Oauth
	if len(config.OauthClients) != 0 {
		Oauth(g, config.OauthClients)
	}

	//Use gzip with level 5
	g.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))

	// injection routers
	if config.Routers != nil {
		config.Routers(g)
	}

	//For HA Health Check
	g.GET("/ping", func(c echo.Context) error {
		return c.JSON(200, "pong")
	})

	return g
}

// 默认错误handler
func DefaultHttpErrorHandler(err error, c echo.Context) {
	var (
		code = http.StatusInternalServerError
		msg  string
	)

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if h, o := he.Message.(string); o {
			msg = h
		} else {
			msg = "Internal Server Error!"
		}
	} else {
		msg = http.StatusText(code)
	}

	if !c.Response().Committed {
		if c.Request().Method == echo.HEAD {
			err := c.NoContent(code)
			if err != nil {
				c.Logger().Error(err)
			}
		} else {
			r := new(Response)
			r.Message = msg
			r.Code = code
			err := c.JSON(200, r)
			if err != nil {
				c.Logger().Error(err)
			}
		}
	}
}

// 开始工作
func (g *Gold) Work(args ...interface{}) error {
	address := ""
	if len(args) > 0 {
		argSlice := args[0].([]interface{})
		if len(argSlice) == 2 {
			ip := argSlice[0]
			port := argSlice[1]
			address = ip.(string) + ":" + port.(string)
		}
	}
	g.Server.Addr = address
	return gracehttp.Serve(g.Server)
}

// 获取配置文件
func (g *Gold) GetConfig() *viper.Viper {
	return g.Configurator
}
