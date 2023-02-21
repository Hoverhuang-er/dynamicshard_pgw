package conf

import (
	"github.com/BurntSushi/toml"
	"log"
	"os"
)

const (
	SDConsul    = 1
	SDEtcd      = 2
	SDNaocs     = 3
	SDZookeeper = 4
	SDRaftd     = 5
)

type Config struct {
	ServiceDiscoveryType int                `yaml,toml:"service_discovery_type"`
	SDServer             *SDServerConfig    `yaml,toml:"sd_server"`
	HttpListenAddr       string             `yaml,toml:"http_listen_addr"`
	PGW                  *PushGateWayConfig `yaml,toml:"pushgateway"`
	Dyshard              *DyshardCfg        `yaml,toml:"dyshard"`
}

type DyshardCfg struct {
	Port int `yaml,toml:"port"`
}

// Use Consul to build hash ring by default
type ConsulServerConfig struct {
	Addr                string `yaml,toml:"addr,omitempty"`
	Username            string `yaml,toml:"username,omitempty"`
	Password            string `yaml,toml:"password,omitempty"`
	RegisterServiceName string `yaml,toml:"register_service_name,omitempty"`
}

// TODO: Also support Etcd, Zookeeper, Nacos or other service discovery work with raft
type SDServerConfig struct {
	ConsulServer *ConsulServerConfig    `yaml,toml:"consul_server"`
	Etcd         *EtcdServerConfig      `yaml,toml:"etcd"`
	Nacos        *NacosServerConfig     `yaml,toml:"nacos"`
	Zookeeper    *ZookeeperServerConfig `yaml,toml:"zookeeper"`
	Raftd        *RaftdServerConfig     `yaml,toml:"raftd"`
}

// Use Etcd to build hash ring
type EtcdServerConfig struct {
	Addr                string `yaml,toml:"addr,omitempty"`
	Username            string `yaml,toml:"username,omitempty"`
	Password            string `yaml,toml:"password,omitempty"`
	RegisterServiceName string `yaml,toml:"register_service_name,omitempty"`
}

// Use Nacos to build hash ring
type NacosServerConfig struct {
	Addr                string `yaml,toml:"addr,omitempty"`
	Username            string `yaml,toml:"username,omitempty"`
	Password            string `yaml,toml:"password,omitempty"`
	RegisterServiceName string `yaml,toml:"register_service_name,omitempty"`
}

// Use Zookeeper to build hash ring
type ZookeeperServerConfig struct {
	Addr                string `yaml,toml:"addr,omitempty"`
	Username            string `yaml,toml:"username,omitempty"`
	Password            string `yaml,toml:"password,omitempty"`
	RegisterServiceName string `yaml,toml:"register_service_name,omitempty"`
}

// Use Build-in Raftd to build hash ring
type RaftdServerConfig struct {
	Addr                string `yaml,toml:"addr,omitempty"`
	RegisterServiceName string `yaml,toml:"register_service_name,omitempty"`
}

type PushGateWayConfig struct {
	Servers []string `yaml,toml:"servers"`
	Port    int      `yaml,toml:"port"`
}

func Load(s string) (*Config, error) {
	cfg := &Config{}

	err := toml.Unmarshal([]byte(s), cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadFile(filename string) (*Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cfg, err := Load(string(content))
	if err != nil {
		log.Printf("load config file %s failed: %v", filename, err)
	}
	log.Print("load config file success")
	return cfg, nil
}

func PathExists(path string) (*Config, error) {
	fi, err := os.Stat(path)
	if err == nil {
		log.Printf("file exists, %v", fi)
		return LoadFile(path)
	} else if os.IsNotExist(err) {
		log.Print("file not exists")
		return nil, err
	} else {
		log.Print("file not exists")
		return nil, err
	}
}
