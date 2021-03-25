package mysqlclient

import (
	"fmt"
	"log"
	"sync"
	"time"

	conf "github.com/elvisNg/broccoliv2/config"
	broccoliprometheus "github.com/elvisNg/broccoliv2/prometheus"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var (
	interceptors []Interceptor
	prom         **broccoliprometheus.Prom
	prometheus   = broccoliprometheus.NewProm()
)

const (
	driverName = "mysql"
	TypeGorm   = "gorm"
)

type DataSource struct {
	Host            string
	User            string
	Pwd             string
	DataSourceName  string
	CharSet         string
	ParseTime       bool
	ConnMaxLifetime time.Duration
	MaxIdleConns    int
	MaxOpenConns    int
}

type Client struct {
	client   *gorm.DB
	rw       sync.RWMutex
	stopChan chan int
}

func newClient() *Client {
	return &Client{
		client:   nil,
		stopChan: make(chan int, 1),
	}
}

func (dbs *Client) Reload(cfg *conf.Mysql) {
	dbs.rw.Lock()
	defer dbs.rw.Unlock()
	dbs.Close()
	log.Printf("[redis.Reload] redisclient reload with new conf: %+v\n", cfg)
	dbs.initClient(cfg)
}

func (dbs *Client) initClient(cfg *conf.Mysql) {
	db := newMysqlClient(cfg, prometheus)
	if db == nil {
		return
	}
	dbs.client = db

	// 监听修复BadConnections
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-ticker.C:
				if err := dbs.client.DB().Ping(); err != nil {
					fmt.Printf("mysql gorm ping err(%+v)\n", err)
				}
			case <-dbs.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (dbs *Client) Close() {
	if err := dbs.client.Close(); err != nil {
		log.Printf("redis close failed: %s\n", err.Error())
		return
	}
}

func InitClientWithProm(sqlconf *conf.Mysql, promClient **broccoliprometheus.Prom) *Client {
	mysql := newClient()
	mysql.initClient(sqlconf)
	prom = promClient
	prometheus = *prom
	return mysql
}

func InitClient(sqlconf *conf.Mysql) *Client {
	mysql := newClient()
	mysql.initClient(sqlconf)
	return mysql
}

func (dbs *Client) GetCli() *gorm.DB {
	return dbs.client
}

func newMysqlClient(cfg *conf.Mysql, prometheus *broccoliprometheus.Prom) *gorm.DB {
	_db, err := open(cfg, prometheus)
	if prometheus != nil {
		interceptors = make([]Interceptor, 0)
		interceptors = append(interceptors, metricInterceptor)
	}
	if err != nil {
		prometheus.Incr(TypeGorm, cfg.DataSourceName+".ping", cfg.Host, "open err")
		fmt.Printf("")
		log.Printf("mysql open err (%+v) , table_(%s)", err.Error(), cfg.DataSourceName+"."+".ping")
		return _db
	}
	return _db
}

// open ... with interceptors
func open(cfg *conf.Mysql, prometheus *broccoliprometheus.Prom) (*gorm.DB, error) {
	url := "%v:%v@(%v)/%v?charset=%v&parseTime=%v&loc=Local"
	//user:password@/dbname?charset=utf8&parseTime=True&loc=Local
	host := cfg.Host
	userName := cfg.User
	passWord := cfg.Pwd
	dbName := cfg.DataSourceName
	charSet := cfg.CharSet
	parseTime := cfg.ParseTime
	url = fmt.Sprintf(url, userName, passWord, host, dbName, charSet, parseTime)
	_db, err := gorm.Open(driverName, url)
	if err != nil {
		println("mysql gorm init err(%+v) ", err)
		panic(fmt.Sprintf("mysql gorm init  failed:%s", err.Error()))
	}

	_db.LogMode(cfg.Debug)

	//全局禁用表名复数
	_db.SingularTable(true) //如果设置为true,`User`的默认表名为`user`,使用`TableName`设置的表名不受影响
	_db.DB().SetMaxOpenConns(cfg.MaxOpenConns)
	_db.DB().SetMaxIdleConns(cfg.MaxIdleConns)
	_db.DB().SetConnMaxLifetime(cfg.ConnMaxLifetime)

	//禁止更新primary key
	if cfg.UpdateSkipPrimaryKey {
		_db.Callback().Update().Before("gorm:assign_updating_attributes").Register("update:skip_primarykey", func(scope *gorm.Scope) {
			pkey := scope.PrimaryKey()
			//println(fmt.Sprintf("primary key:'%s'", pkey))
			if len(pkey) > 0 {
				omitAttrs := scope.OmitAttrs()
				omitAttrs = append(omitAttrs, pkey)
				scope.Search = scope.Search.Omit(omitAttrs...)
			}
		})
	}
	if prometheus == nil {
		return _db, err
	}

	replace := func(processor func() *gorm.CallbackProcessor, callbackName string, interceptors ...Interceptor) {
		old := processor().Get(callbackName)
		var handler = old
		for _, inte := range interceptors {
			handler = inte(cfg, prometheus)(handler)
		}
		processor().Replace(callbackName, handler)
	}

	replace(
		_db.Callback().Delete,
		"gorm:delete",
		interceptors...,
	)
	replace(
		_db.Callback().Update,
		"gorm:update",
		interceptors...,
	)
	replace(
		_db.Callback().Create,
		"gorm:create",
		interceptors...,
	)
	replace(
		_db.Callback().Query,
		"gorm:query",
		interceptors...,
	)
	replace(
		_db.Callback().RowQuery,
		"gorm:row_query",
		interceptors...,
	)
	return _db, err
}
