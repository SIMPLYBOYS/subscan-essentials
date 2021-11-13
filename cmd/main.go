package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/CoolBitX-Technology/subscan/configs"
	"github.com/CoolBitX-Technology/subscan/internal/script"
	"github.com/CoolBitX-Technology/subscan/internal/server/http/handler"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/pkg/conf/paladin"
	"github.com/itering/substrate-api-rpc/websocket"
	"github.com/prometheus/common/log"
	"github.com/urfave/cli"
)

func main() {
	defer func() {
		websocket.Close()
	}()
	if err := setupApp().Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setupApp() *cli.App {
	observer := NewObserver()
	app := cli.NewApp()
	app.Name = "SUBSCAN"
	app.Usage = "SUBSCAN Backend Service, use -h get help"
	app.Version = "1.0"
	app.Action = func(*cli.Context) error { run(); return nil }
	app.Description = "SubScan Backend Service, substrate blockchain explorer"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "conf", Value: "./configs"},
	}
	app.Before = func(context *cli.Context) error {
		if client, err := paladin.NewFile(context.String("conf")); err != nil {
			panic(err)
		} else {
			paladin.DefaultClient = client
		}
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:  "start",
			Usage: "Start one worker, E.g substrate",
			Action: func(c *cli.Context) error {
				observer.Run(c.Args(), "start")
				return nil
			},
		},
		{
			Name:  "stop",
			Usage: "Stop one worker, E.g substrate",
			Action: func(c *cli.Context) error {
				observer.Run(c.Args(), "stop")
				return nil
			},
		},
		{
			Name:  "install",
			Usage: "Create database and create default conf file",
			Action: func(c *cli.Context) error {
				script.Install(c.Parent().String("conf"))
				return nil
			},
		},
	}
	return app
}

func run() {
	websocket.SetEndpoint(util.WSEndPoint)
	conn, err := websocket.Init()

	for i := 0; i < retry; i++ {
		if !conn.Conn.IsConnected() {
			conn.Conn.Dial(util.WSEndPoint, nil)
		} else {
			break
		}
		fmt.Fprintf(os.Stderr, "websocket dial error: %+v\n", err)
		fmt.Fprintf(os.Stderr, "%d times Retrying in %v\n", i+1, 10*time.Second)
		time.Sleep(10 * time.Second)
	}

	ds, err := initDS()

	if err != nil {
		log.Error("Unable to initialize data sources: ", err)
	}

	err, common, block, extrinsic, event, runtime, plugin, _, _, _, _ := inject(ds)

	if err != nil {
		log.Error("Failure to inject data sources: ", err)
	}

	common.InitSubRuntimeLatest()
	plugin.PluginRegister()
	// gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	handler.NewHandler(&handler.Config{
		R:                router,
		CommonService:    common,
		BlockService:     block,
		ExtrinsicService: extrinsic,
		EventService:     event,
		RuntimeService:   runtime,
	})

	var hc configs.HttpConf
	hc.MergeConf()

	srv := &http.Server{
		Addr:    hc.Server.Addr,
		Handler: router,
	}

	c := make(chan os.Signal, 1)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to initialize server: ", err)
		}
	}()

	log.Info("Listening on port ", srv.Addr)

	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		log.Info("get a signal ", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
			defer cancel()
			common.Close()
			if err := srv.Shutdown(ctx); err != nil {
				log.Error("httpSrv.Shutdown error(", err, ")")
			}
			log.Info("SubScan End exit")
			time.Sleep(time.Second)
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}
