package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/itering/substrate-api-rpc/pkg/recws"
	"github.com/itering/substrate-api-rpc/websocket"
	"github.com/prometheus/common/log"
	"github.com/sevlyar/go-daemon"
)

type observer struct {
}

type ObserverInterface interface {
	Run(dt []string, signal string)
}

func NewObserver() ObserverInterface {
	return &observer{}
}

func (o *observer) Run(dt []string, signal string) {
	// daemon.AddCommand(daemon.StringFlag(&signal, "stop"), syscall.SIGQUIT, termHandler)
	doAction(dt)
}

func doAction(dt []string) {
	if !util.StringInSlice(dt[0], model.DaemonAction) {
		log.Info("no such daemon")
		return
	}

	// logDir := util.GetEnv("LOG_DIR", "../log/")
	// pid := fmt.Sprintf("%s%s_pid", logDir, dt)
	// // logName := fmt.Sprintf("%s%s_log", logDir, dt)

	// dc := &daemon.Context{
	// 	PidFileName: pid,
	// 	PidFilePerm: 0644,
	// 	LogFileName: "/dev/stdout",
	// 	// LogFileName: logName,
	// 	LogFilePerm: 0640,
	// 	WorkDir:     "./",
	// 	Umask:       027,
	// 	Args:        nil,
	// }

	// if len(daemon.ActiveFlags()) > 0 {
	// 	d, err := dc.Search()
	// 	if err != nil {
	// 		log.Info(dt, "not running")
	// 	} else {
	// 		_ = daemon.SendCommands(d)
	// 	}
	// 	return
	// }

	// // d, err := dc.Reborn()
	// // if err != nil {
	// // 	log.Fatalln(err)
	// // }
	// // if d != nil {
	// // 	return
	// // }
	// defer func() {
	// 	err := dc.Release()
	// 	if err != nil {
	// 		log.Info("Error:", err)
	// 	}
	// }()

	// log.Info("- - - - - - - - - - - - - - -")
	// log.Info("daemon started")

	go doRun(dt)

	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-sigs
		log.Info("get a signal ", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			log.Info("daemon terminated")
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}

	// err := daemon.ServeSignals()
	// if err != nil {
	// 	log.Info("Error:", err)
	// }
}

var (
	stop      = make(chan struct{})
	done      = make(chan struct{})
	subscribe model.SubscribeService
	repair    model.RepairService
	cache     model.RedisRepository // alias redis
	sql       model.SqlRepository
	common    model.CommonService
	plugin    model.PluginService
	sigs      = make(chan os.Signal, 1)
)

func doRun(dt []string) {
	websocket.SetEndpoint(util.WSEndPoint)
	conn, err := websocket.Init()

	for i := 0; i < retry; i++ {
		if !conn.Conn.IsConnected() {
			conn.Conn.Dial(util.WSEndPoint, nil)
		} else {
			break
		}
		fmt.Fprintf(os.Stderr, "websocket dial error: %+v\n", err)
		fmt.Fprintf(os.Stderr, "%d times Retrying in %s\n", i+1, 10*time.Second)
		time.Sleep(10 * time.Second)
	}

	ds, err := initDS()
	if err != nil {
		log.Fatal("Unable to initialize data sources:", err, "\n")
	}
	_, common, _, _, _, _, plugin, subscribe, repair, cache, sql = inject(ds) // TODO repair service
	blockNum, _ := cache.GetFillBestBlockNum(context.TODO())
	sql.Migration(blockNum)
	common.InitSubRuntimeLatest()
	plugin.PluginRegister()
	defer cache.Close()
LOOP:
	for {
		if dt[0] == "substrate" || dt[0] == "plugins" || dt[0] == "repair" {
			interrupt := make(chan os.Signal, 1)
			subscribeConn := &recws.RecConn{KeepAliveTimeout: 60 * time.Second, WriteTimeout: time.Second * 30, ReadTimeout: 30 * time.Second}
			subscribeConn.Dial(util.WSEndPoint, nil)
			log.Info("Dial to :", util.WSEndPoint)
			switch dt[0] {
			case "substrate":
				subscribe.Subscribe(subscribeConn, interrupt)
			case "plugins":
				plugin.Subscribe(subscribeConn, interrupt)
			case "repair":
				head, _ := strconv.Atoi(dt[1])
				size, _ := strconv.Atoi(dt[2])
				repair.Repair(subscribeConn, interrupt, head, size)
			}
		} else {
			go heartBeat(dt[0])
			switch dt {
			default:
				break LOOP
			}
		}

		if _, ok := <-stop; ok {
			break LOOP
		}
	}
	done <- struct{}{}
}

func termHandler(sig os.Signal) error {
	log.Info("terminating...")
	stop <- struct{}{}
	if sig == syscall.SIGQUIT {
		<-done
	}
	return daemon.ErrStop
}

func heartBeat(dt string) {
	for {
		common.SetHeartBeat(fmt.Sprintf("%s:heartBeat:%s", util.NetworkNode, dt))
		time.Sleep(10 * time.Second)
	}
}
