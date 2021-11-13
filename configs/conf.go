package configs

import (
	"fmt"

	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/go-kratos/kratos/pkg/conf/paladin"
	"github.com/go-kratos/kratos/pkg/database/sql"
)

type (
	MysqlConf struct {
		Conf struct {
			Host string
			User string
			Pass string
			DB   string
		}
		Api  *sql.Config
		Task *sql.Config
		Test *sql.Config
	}
	HttpConf struct {
		Server struct {
			Addr    string
			Timeout string
		}
	}
	RedisConf struct {
		Config struct {
			Addr       string
			Addrs      string
			DbName     int
			MasterName string
			AuthPw     string
			Pw         string
		}
	}
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func (dc *MysqlConf) MergeConf() {
	checkErr(paladin.Get("mysql.toml").UnmarshalTOML(dc))
	dc.mergeEnvironment()
}

func (rc *RedisConf) MergeConf() {
	checkErr(paladin.Get("redis.toml").UnmarshalTOML(rc))
	rc.mergeEnvironment()
}

func (hc *HttpConf) MergeConf() {
	checkErr(paladin.Get("http.toml").UnmarshalTOML(hc))
	hc.mergeEnvironment()
}

func (dc *MysqlConf) mergeEnvironment() {
	dbHost := util.GetEnv("MYSQL_HOST", dc.Conf.Host)
	dbUser := util.GetEnv("MYSQL_USER", dc.Conf.User)
	dbPass := util.GetEnv("MYSQL_PASS", dc.Conf.Pass)
	dbName := fmt.Sprintf("%s", util.GetEnv("MYSQL_DB", "subscan"))
	dc.Api.DSN = fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPass, dbHost, dbName) + dc.Api.DSN
	dc.Task.DSN = fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPass, dbHost, dbName) + dc.Task.DSN
}

func (rc *RedisConf) mergeEnvironment() {
	rc.Config.Addr = util.GetEnv("REDIS_ADDR", rc.Config.Addr)
	rc.Config.Addrs = util.GetEnv("SENTINEL_ADDRS", rc.Config.Addrs) // host_a:1234,host_b:4321
	rc.Config.DbName = util.StringToInt(util.GetEnv("REDIS_DATABASE", "0"))
	rc.Config.MasterName = util.GetEnv("REDIS_MASTER_NAME", rc.Config.MasterName)
	rc.Config.Pw = util.GetEnv("REDIS_PW", rc.Config.Pw)
	rc.Config.AuthPw = util.GetEnv("REDIS_AUTH_PW", rc.Config.AuthPw)
}

func (hc *HttpConf) mergeEnvironment() {
	hc.Server.Addr = util.GetEnv("HTTP_ADDR", hc.Server.Addr)
	hc.Server.Timeout = util.GetEnv("HTTP_TIMEOUT", "10s")
}
