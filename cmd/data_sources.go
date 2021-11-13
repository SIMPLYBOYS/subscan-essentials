package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CoolBitX-Technology/subscan/configs"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/go-kratos/kratos/pkg/log"
	"github.com/go-redis/redis/v8"
	"github.com/jinzhu/gorm"
)

type dataSources struct {
	DB    *gorm.DB
	Redis *redis.Client
}

type ormLog struct{}

func (l ormLog) Print(v ...interface{}) {
	log.Info(strings.Repeat("%v ", len(v)), v...)
}

const retry = 10

func initDS() (*dataSources, error) {
	log.Info("== initDS ==")
	var dc configs.MysqlConf
	var rc configs.RedisConf
	var rs *redis.Client
	var err error

	dc.MergeConf()
	rc.MergeConf()
	db := newDb(dc)

	for i := 0; i < retry; i++ {
		if util.IsProduction {
			addrs := strings.Split(rc.Config.Addrs, ",")
			rs = redis.NewFailoverClient(&redis.FailoverOptions{
				MasterName:       rc.Config.MasterName,
				SentinelAddrs:    addrs,
				SentinelPassword: rc.Config.AuthPw,
				Password:         rc.Config.Pw,
				DB:               rc.Config.DbName,
			})
		} else {
			rs = redis.NewClient(&redis.Options{
				Addr:     rc.Config.Addr,
				Password: rc.Config.Pw,
				DB:       rc.Config.DbName,
			})
		}
		// verify redis connection
		_, err = rs.Ping(context.Background()).Result()

		if err == nil {
			break
		}
		fmt.Fprintf(os.Stderr, "Redis request error: %+v\n", err)
		fmt.Fprintf(os.Stderr, "%d times Retrying in %v\n", i+1, 10*time.Second)
		time.Sleep(10 * time.Second)
	}

	if err != nil {
		return nil, err
	}
	return &dataSources{DB: db, Redis: rs}, nil
}

func newDb(dc configs.MysqlConf) (db *gorm.DB) {
	var err error
	for i := 0; i < retry; i++ {
		if os.Getenv("TASK_MOD") == "true" {
			db, err = gorm.Open("mysql", dc.Task.DSN)
		} else if os.Getenv("TEST_MOD") == "true" {
			db, err = gorm.Open("mysql", dc.Test.DSN)
		} else {
			db, err = gorm.Open("mysql", dc.Api.DSN)
		}

		if err == nil {
			break
		}
		fmt.Fprintf(os.Stderr, "MySql Request error: %+v\n", err)
		fmt.Fprintf(os.Stderr, "%d times Retrying in %v\n", i+1, 10*time.Second)
		time.Sleep(10 * time.Second)
	}

	if err != nil {
		panic(err)
	}

	db.DB().SetConnMaxLifetime(5 * time.Minute)
	db.DB().SetMaxOpenConns(100)
	db.DB().SetMaxIdleConns(10)
	if util.IsProduction {
		db.SetLogger(ormLog{})
	}
	if os.Getenv("TEST_MOD") != "true" {
		db.LogMode(false)
	}
	return db
}
