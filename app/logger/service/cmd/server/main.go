package main

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"

	// etcd config
	etcdKratos "github.com/go-kratos/kratos/contrib/config/etcd/v2"
	etcdV3 "go.etcd.io/etcd/client/v3"
	GRPC "google.golang.org/grpc"

	// consul config
	consulKratos "github.com/go-kratos/kratos/contrib/config/consul/v2"
	"github.com/hashicorp/consul/api"

	// nacos config
	nacosKratos "github.com/go-kratos/kratos/contrib/config/nacos/v2"
	nacosClients "github.com/nacos-group/nacos-sdk-go/clients"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"kratos-cqrs/app/logger/service/internal/conf"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	Name          = "kratos.logger.service"
	Version       = "1.0.0"
	InstanceId, _ = os.Hostname()

	flagConf       string
	flagEnv        string
	flagConfigHost string
	flagConfigType string
)

func init() {
	flag.StringVar(&flagConf, "conf", "../../configs", "config path, eg: -conf config.yaml")
	flag.StringVar(&flagEnv, "env", "dev", "runtime environment, eg: -env dev")
	flag.StringVar(&flagConfigHost, "chost", "127.0.0.1:8500", "config server host, eg: -chost 127.0.0.1:8500")
	flag.StringVar(&flagConfigType, "ctype", "consul", "config server host, eg: -ctype consul")
}

func newApp(logger log.Logger, gs *grpc.Server, rr registry.Registrar) *kratos.App {
	return kratos.New(
		kratos.ID(InstanceId+"."+Name),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
		),
		kratos.Registrar(rr),
	)
}

func NewTracerProvider(conf *conf.Trace) error {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(conf.Endpoint)))
	if err != nil {
		return err
	}

	tp := traceSdk.NewTracerProvider(
		traceSdk.WithSampler(traceSdk.ParentBased(traceSdk.TraceIDRatioBased(1.0))),
		traceSdk.WithBatcher(exp),
		traceSdk.WithResource(resource.NewSchemaless(
			semConv.ServiceNameKey.String(Name),
			semConv.ServiceVersionKey.String(Version),
			semConv.ServiceInstanceIDKey.String(InstanceId),
			attribute.String("flagEnv", flagEnv),
		)),
	)

	otel.SetTracerProvider(tp)

	return nil
}

func NewLoggerProvider() log.Logger {
	l := log.NewStdLogger(os.Stdout)
	return log.With(
		l,
		"service.InstanceId", InstanceId,
		"service.name", Name,
		"service.version", Version,
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"trace_id", tracing.TraceID(),
		"span_id", tracing.SpanID(),
	)
}

func getConfigKey(useBackslash bool) string {
	if useBackslash {
		return strings.Replace(Name, `.`, `/`, -1)
	} else {
		return Name
	}
}

func NewRemoteConfigSource() config.Source {
	switch flagConfigType {
	case "nacos":
		return NewNacosConfigSource()
	case "consul":
		return NewConsulConfigSource()
	case "etcd":
		return NewEtcdConfigSource()
	case "apollo":
		return NewApolloConfigSource()
	}
	return nil
}

func NewNacosConfigSource() config.Source {
	sc := []nacosConstant.ServerConfig{
		*nacosConstant.NewServerConfig("127.0.0.1", 8849),
	}

	cc := nacosConstant.ClientConfig{
		TimeoutMs:            10 * 1000, // http请求超时时间，单位毫秒
		BeatInterval:         5 * 1000,  // 心跳间隔时间，单位毫秒
		UpdateThreadNum:      20,        // 更新服务的线程数
		LogLevel:             "debug",
		CacheDir:             "../../configs/cache", // 缓存目录
		LogDir:               "../../configs/log",   // 日志目录
		NotLoadCacheAtStart:  true,                  // 在启动时不读取本地缓存数据，true--不读取，false--读取
		UpdateCacheWhenEmpty: true,                  // 当服务列表为空时是否更新本地缓存，true--更新,false--不更新
	}

	nacosClient, err := nacosClients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)
	if err != nil {
		panic(err)
	}

	return nacosKratos.NewConfigSource(nacosClient,
		nacosKratos.WithGroup(getConfigKey(false)),
		nacosKratos.WithDataID("config.yaml"),
	)
}

func NewEtcdConfigSource() config.Source {
	etcdClient, err := etcdV3.New(etcdV3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: time.Second, DialOptions: []GRPC.DialOption{GRPC.WithBlock()},
	})
	if err != nil {
		panic(err)
	}

	etcdSource, err := etcdKratos.New(etcdClient, etcdKratos.WithPath(getConfigKey(true)))
	if err != nil {
		panic(err)
	}

	return etcdSource
}

func NewApolloConfigSource() config.Source {
	return nil
}

func NewConsulConfigSource() config.Source {
	consulClient, err := api.NewClient(&api.Config{
		Address: flagConfigHost,
	})
	if err != nil {
		panic(err)
	}

	consulSource, err := consulKratos.New(consulClient, consulKratos.WithPath(getConfigKey(true)))
	if err != nil {
		panic(err)
	}

	//w, err := consulSource.Watch()
	//if err != nil {
	//	panic(err)
	//}

	return consulSource
}

func NewFileConfigSource() config.Source {
	return file.NewSource(flagConf)
}

func NewConfigProvider() config.Config {
	return config.New(
		config.WithSource(
			NewFileConfigSource(),
			NewRemoteConfigSource(),
		),
	)
}

func loadConfig() (*conf.Bootstrap, *conf.Registry) {
	c := NewConfigProvider()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	var rc conf.Registry
	if err := c.Scan(&rc); err != nil {
		panic(err)
	}

	return &bc, &rc
}

func main() {
	flag.Parse()

	logger := NewLoggerProvider()

	bc, rc := loadConfig()
	if bc == nil || rc == nil {
		panic("load config failed")
	}

	err := NewTracerProvider(bc.Trace)
	if err != nil {
		panic(err)
	}

	app, cleanup, err := initApp(bc.Server, rc, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
