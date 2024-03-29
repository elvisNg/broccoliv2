package zgomicro

import (
	"context"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/grpc"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-plugins/registry/etcdv3"

	"github.com/elvisNg/broccoliv2/config"
)

func NewClient(ctx context.Context, conf config.GoMicro, opts ...client.Option) (cli client.Client, err error) {
	// etcd registry
	var reg registry.Registry
	switch conf.RegistryPluginType {
	case "etcd":
		reg = etcdv3.NewRegistry(
			registry.Addrs(conf.RegistryAddrs...),
			etcdv3.Auth(conf.RegistryAuthUser, conf.RegistryAuthPwd),
		)
	default:
		reg = registry.DefaultRegistry
	}
	o := []client.Option{
		client.Registry(reg),
		client.PoolSize(conf.ClientPoolSize),
		client.PoolTTL(100 * time.Second),
		client.RequestTimeout(30 * time.Second),
		grpc.MaxRecvMsgSize(1024 * 1024 * 10),
		grpc.MaxSendMsgSize(1024 * 1024 * 10),
	}
	o = append(o, opts...)
	// new client
	cli = grpc.NewClient(o...)
	if err = cli.Init(); err != nil {
		return nil, err
	}
	return
}
