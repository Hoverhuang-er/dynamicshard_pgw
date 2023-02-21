package method

import (
	"context"
	"fmt"
	"github.com/Hoverhuang-er/dynamicshard/pkg/conf"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type ConsistConsul struct {
	Addr                string                  `yaml,toml:"addr,omitempty"`
	UserName            string                  `yaml,toml:"username,omitempty"`
	Password            string                  `yaml,toml:"password,omitempty"`
	RegisterServiceName string                  `yaml,toml:"register_service_name,omitempty"`
	PGW                 *conf.PushGateWayConfig `yaml,toml:"pushgateway"`
	ServerCfg           *conf.DyshardCfg        `yaml,toml:"dyshard"`
}

func UseConsulSD(cfg *conf.Config) error {
	cc := &ConsistConsul{
		Addr:                cfg.SDServer.ConsulServer.Addr,
		UserName:            cfg.SDServer.ConsulServer.Username,
		Password:            cfg.SDServer.ConsulServer.Password,
		RegisterServiceName: cfg.SDServer.ConsulServer.RegisterServiceName,
		PGW:                 cfg.PGW,
		ServerCfg:           cfg.Dyshard,
	}
	return cc.NewConsistentRing()
}

func (cc *ConsistConsul) NewConsistentRing() error {
	// new grpc manager
	ctx, cancelT := context.WithCancel(context.Background())
	defer cancelT()
	csc, err := NewConsulClient(cc.Addr)
	if err != nil {
		return err
	}
	var ss []string
	for _, i := range cc.PGW.Servers {
		ss = append(ss, fmt.Sprintf("%s:%d", i, cc.PGW.Port))
	}
	errors := RegisterFromFile(csc, cc.PGW.Servers, cc.RegisterServiceName, cc.PGW.Port)
	if len(errors) > 0 {
		return errors[0]
	}
	NewConsistentHashNodesRing(ss)
	var g run.Group
	{
		// Termination handler.
		term := make(chan os.Signal, 1)
		signal.Notify(term, os.Interrupt, syscall.SIGTERM)
		cancel := make(chan struct{})
		g.Add(

			func() error {
				select {
				case <-term:
					cancelT()
					return nil
					//TODO clean work here
				case <-cancel:
					return nil
				}
			},
			func(err error) {
				close(cancel)

			},
		)
	}
	{
		// WatchService   manager.
		g.Add(func() error {
			err := csc.RunRefreshServiceNode(context.Background(), cc.RegisterServiceName, cc.Addr)
			if err != nil {
			}
			return err
		}, func(err error) {
			cancelT()
		})
	}
	{
		g.Add(func() error {
			// start chi server
			ch := chi.NewMux()
			ch.Get("/prom", promhttp.Handler().ServeHTTP)
			ch.Route("/metrics/job", func(cu chi.Router) {
				cu.Get("/*any", PushMetricsGetHashV2)
				cu.Put("/*any", PushMetricsRedirectV2)
				cu.Post("/*any", PushMetricsRedirectV2)
			})
			ch.Route("/test/", func(ct chi.Router) {
				ct.Get("/v1", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Hello, I'm pgw gateway+ (｡A｡"))
					return
				})
			})
			errchan := make(chan error, 1)
			go func() {
				errchan <- http.ListenAndServe(fmt.Sprintf(":%d", cc.ServerCfg.Port), ch)
			}()
			select {
			case err := <-errchan:
				return err
			case <-ctx.Done():
				return nil

			}
		}, func(err error) {
			cancelT()
		})
	}
	g.Run()
	return nil
}
