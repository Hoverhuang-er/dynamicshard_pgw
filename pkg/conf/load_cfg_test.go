package conf

import (
	"github.com/BurntSushi/toml"
	"os"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		// TODO: Add test cases .
		{
			name: "test yaml",
			args: args{
				s: ``,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test yaml",
			args: args{
				filename: "config.toml",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadFile(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadFile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathExists(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test.",
			args: args{
				path: "config.toml",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PathExists(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("PathExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PathExists() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenCfg(t *testing.T) {
	f, err := os.OpenFile("config.dev.toml", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(&Config{
		ServiceDiscoveryType: 1,
		SDServer: &SDServerConfig{
			ConsulServer: &ConsulServerConfig{
				Addr:                "127.0.0.1:8500",
				Username:            "admin",
				Password:            "password",
				RegisterServiceName: "consistentring",
			},
			Etcd:      nil,
			Nacos:     nil,
			Zookeeper: nil,
			Raftd:     nil,
		},
		HttpListenAddr: "8083",
		PGW: &PushGateWayConfig{
			Servers: []string{"172.17.0.1"},
			Port:    9091,
		},
		Dyshard: &DyshardCfg{
			Port: 8082,
		},
	}); err != nil {
		t.Error(err)
		return
	}
}
