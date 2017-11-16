package service

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/uber-go/zap"
)

type Center struct {
}

func New() *Center {
	return &Center{}
}

func (c *Center) Start(isStatic bool) {
	loadConfig(isStatic)

	initMysql()
	go httpStart()

}

func httpStart() {
	// 启动Http服务
	e := echo.New()

	// 启动配置热更新
	e.GET("/reload", reload)

	// appid管理
	e.POST("/appid/manage", appidManage)

	err := e.Start(":4001")
	if err != nil {
		Logger.Fatal("echo http start failed", zap.Error(err))
	}
}

var db *sql.DB

func initMysql() {
	var err error

	// 初始化mysql连接
	sqlConn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", Conf.Mysql.Acc, Conf.Mysql.Pw,
		Conf.Mysql.Addr, Conf.Mysql.Port, Conf.Mysql.Database)
	db, err = sql.Open("mysql", sqlConn)
	if err != nil {
		Logger.Fatal("[FATAL] init mysql error", zap.Error(err))
	}

	// 测试db是否正常
	err = db.Ping()
	if err != nil {
		Logger.Fatal("[FATAL] ping mysql error", zap.Error(err))
	}
}
