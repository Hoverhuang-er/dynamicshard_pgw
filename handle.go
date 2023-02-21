package main

import (
	"dynamicshard/pkg/conf"
	"dynamicshard/pkg/method"
	"log"
	"os"
)

func main() {
	cfg, err := conf.PathExists("config.toml")
	if err != nil {
		log.Printf("load config file %s failed: %v", "config.toml", err)
		return
	}
	log.Print("load config file success")
	switch cfg.ServiceDiscoveryType {
	case conf.SDConsul:
		log.Print("consul")
		log.Print("Currently support consul service discovery")
		if err := method.UseConsulSD(cfg); err != nil {
			log.Printf("use consul service discovery failed: %v", err)
			return
		}
	case conf.SDEtcd:
		log.Print("etcd")
	case conf.SDNaocs:
		log.Print("nacos")
	case conf.SDZookeeper:
		log.Print("zookeeper")
	case conf.SDRaftd:
		log.Print("raftd")
	default:
		log.Print("sd not found")
	}
	os.Exit(127)
}
