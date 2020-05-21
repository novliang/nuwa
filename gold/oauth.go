package gold

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/novliang/nuwa/utils"
	"gopkg.in/oauth2.v3"
	"gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/generates"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/server"
	"gopkg.in/oauth2.v3/store"
	"log"
	"time"
	error2 "errors"
)

type OauthClient struct {
	ClientName           string
	ClientStore          oauth2.ClientStore
	AuthorizationHandler func(username, password string) (userID string, err error)
}

var Servers = make(map[string]*server.Server)

// 加载认证
func Oauth(g *Gold, clients []*OauthClient) {

	jwtConfig := g.Configurator.GetStringMap("jwt")
	if len(jwtConfig) == 0 {
		panic("can't fin jwt config")
	}

	for _, v := range clients {
		// 配置Server 和 Manager
		m := manage.NewDefaultManager()
		m.MustTokenStorage(store.NewFileTokenStore(utils.GetAppPath() + "/" + v.ClientName))
		m.MapClientStorage(v.ClientStore)
		m.MapAccessGenerate(generates.NewJWTAccessGenerate([]byte(jwtConfig["secret"].(string)), jwt.SigningMethodHS512))

		s := server.NewDefaultServer(m)
		s.SetAllowGetAccessRequest(true)
		s.SetClientInfoHandler(server.ClientFormHandler)
		s.SetPasswordAuthorizationHandler(v.AuthorizationHandler)
		s.SetInternalErrorHandler(func(err error) (re *errors.Response) {
			log.Println("Internal Error:", err.Error())
			return
		})
		s.SetResponseErrorHandler(func(re *errors.Response) {
			log.Println("Response Error:", re.Error.Error())
		})
		Servers[v.ClientName] = s
	}

	g.GET("/token", func(c echo.Context) error {
		cx := &Context{c}
		sh := c.Request().Header.Get("NUWA-OAUTH-SERVER")
		if s, ok := Servers[sh]; ok {
			gt, tgr, err := s.ValidationTokenRequest(cx.Request())
			if err != nil {
				return echo.NewHTTPError(400, err.Error())
			}

			t, _ := time.ParseDuration("1440h")
			tgr.AccessTokenExp = t
			ti, err := s.GetAccessToken(gt, tgr)
			if err != nil {
				return echo.NewHTTPError(400, err.Error())
			}
			return cx.Out(s.GetTokenData(ti))
		}
		return cx.OutEmptyMap()
	})
}

// 守卫
func Guard(jwtConfig map[string]interface{}) echo.MiddlewareFunc {
	return func(handlerFunc echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sh := c.Request().Header.Get("NUWA-OAUTH-SERVER")
			if s, ok := Servers[sh]; ok {
				access, ok := s.BearerAuth(c.Request())
				if !ok {
					return echo.NewHTTPError(401, errors.ErrInvalidAccessToken.Error())
				}
				// Parse and verify jwt access token
				token, err := jwt.ParseWithClaims(access, &generates.JWTAccessClaims{}, func(t *jwt.Token) (interface{}, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("parse error")
					}
					return []byte(jwtConfig["secret"].(string)), nil
				})
				if err != nil {
					return echo.NewHTTPError(401, errors.ErrInvalidAccessToken.Error())
				}

				_, ok = token.Claims.(*generates.JWTAccessClaims)
				if !ok || !token.Valid {
					return echo.NewHTTPError(401, errors.ErrInvalidAccessToken.Error())
				}
			} else {
				return echo.NewHTTPError(401, errors.ErrInvalidAccessToken.Error())
			}
			return handlerFunc(c)
		}
	}
}

// 获取客户端ID
func GetClientId(cx *Context) (string, error) {
	// 设置分组ID
	sh := cx.Request().Header.Get("NUWA-OAUTH-SERVER")
	if s, ok := Servers[sh]; ok {
		token, err := s.ValidationBearerToken(cx.Request())
		if err != nil {
			return "", err
		}
		groupId := token.GetClientID()
		if groupId == "" {
			return "", error2.New("获取Client失败")
		}

		return groupId, nil
	}
	return "", error2.New("获取Client失败")
}

// 获取UserID
func GetUserId(cx *Context) (string, error) {
	// 设置分组ID
	sh := cx.Request().Header.Get("NUWA-OAUTH-SERVER")
	if s, ok := Servers[sh]; ok {
		token, err := s.ValidationBearerToken(cx.Request())
		if err != nil {
			return "", err
		}
		userId := token.GetUserID()
		if userId == "" {
			return "", error2.New("获取用户ID失败")
		}
		return userId, nil
	}
	return "", error2.New("获取用户ID失败")
}
