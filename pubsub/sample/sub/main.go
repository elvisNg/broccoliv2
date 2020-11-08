package main

import (
	"context"
	"log"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/micro/go-micro/metadata"

	"github.com/elvisNg/broccoliv2/config"
	brokerpb "github.com/elvisNg/broccoliv2/pubsub/proto"
	zsub "github.com/elvisNg/broccoliv2/pubsub/sub"
	zprotobuf "github.com/elvisNg/broccoliv2/utils/protobuf"
)

func main() {
	// 默认数据源(单个)
	sub()
	// 多个数据源
	// subManager()
}

// sub 默认数据源(单个)
func sub() {
	conf := &config.Broker{}

	// conf.Type = "redis"
	// conf.TopicPrefix = "dev"

	conf.Type = "kafka"
	conf.TopicPrefix = "dev"
	conf.Hosts = []string{"10.1.8.14:9094"}

	conf.SubscribeTopics = append(conf.SubscribeTopics, &config.TopicInfo{Category: "sample", Source: "broccoli", Queue: "cache"})
	conf.SubscribeTopics = append(conf.SubscribeTopics, &config.TopicInfo{Category: "samplerequest", Source: "broccoli", Queue: "cache"})
	conf.SubscribeTopics = append(conf.SubscribeTopics, &config.TopicInfo{Category: "pbstruct", Source: "broccoli", Queue: "cache"})
	conf.SubscribeTopics = append(conf.SubscribeTopics, &config.TopicInfo{Category: "jsonrequest", Source: "broccoli", Queue: "cache"})
	zsub.InitDefault(conf)
	handlers := make(map[string]interface{})
	handlers["sample.broccoli"] = SampleHandler
	handlers["samplerequest.broccoli"] = SampleRequestHandler
	handlers["pbstruct.broccoli"] = PBStructHandler
	handlers["jsonrequest.broccoli"] = JSONRequestHandler
	zsub.Subscribe(context.Background(), handlers)
	if err := zsub.Run(context.Background()); err != nil {
		log.Println(err)
		return
	}
}

// subManager 多个数据源
func subManager() {
	brokerSource := map[string]config.Broker{
		"broccoli": config.Broker{
			Type:        "kafka",
			TopicPrefix: "dev",
			Hosts:       []string{"10.1.8.14:9094"},
			SubscribeTopics: []*config.TopicInfo{
				&config.TopicInfo{Category: "sample", Source: "broccoli", Queue: "cache-broccoli"},
				&config.TopicInfo{Category: "pbstruct", Source: "broccoli", Queue: "cache-broccoli"},
			},
		},
		"dzqz": config.Broker{
			Type:        "kafka",
			TopicPrefix: "dev",
			Hosts:       []string{"10.1.8.14:9094"},
			SubscribeTopics: []*config.TopicInfo{
				&config.TopicInfo{Category: "samplerequest", Source: "broccoli", Queue: "cache-dzqz"},
				&config.TopicInfo{Category: "jsonrequest", Source: "broccoli", Queue: "cache-dzqz"},
			},
		},
		"broccoli-rb": config.Broker{
			Type:  "rabbitmq",
			Hosts: []string{"amqp://guest:guest@127.0.0.1:5672"},
			// ExchangeName: "broccoli",
			// ExchangeKind: "direct",
			SubscribeTopics: []*config.TopicInfo{
				&config.TopicInfo{Topic: "pbstruct.broccoli", Queue: "cache-broccoli-rb"}, // 这里的Topic 是作为 routing-key
			},
		},
	}
	handlers := map[string]map[string]interface{}{
		"broccoli": map[string]interface{}{
			"sample.broccoli":   SampleHandler,
			"pbstruct.broccoli": PBStructHandler,
		},
		"dzqz": map[string]interface{}{
			"samplerequest.broccoli": SampleRequestHandler,
			"jsonrequest.broccoli":   JSONRequestHandler,
		},
		"broccoli-rb": map[string]interface{}{
			"pbstruct.broccoli": RawDataHandler,
		},
	}
	confList := make(map[string]*zsub.SubConfig)
	for s, b := range brokerSource {
		// 此处需要注意：b为结构体类型，不能直接指针&b赋值，需要使用中间变量复制b结构体值
		tmp := b
		sc := &zsub.SubConfig{
			BrokerConf: &tmp,
			Handlers:   handlers[s],
		}
		confList[s] = sc
	}
	mc := &zsub.ManagerConfig{
		Conf: confList,
		// JSONCodecFn: NewCodec, // 可使用自定义实现的json Codec覆盖
	}
	m, err := zsub.NewManager(mc)
	if err != nil {
		log.Println(err)
		return
	}
	if err = m.Subscribe(context.Background()); err != nil {
		log.Println(err)
		return
	}
	m.Run(context.Background())
}

func SampleHandler(ctx context.Context, msg *brokerpb.Sample) error {
	m, _ := metadata.FromContext(ctx)
	log.Println("metadata", m)
	log.Printf("SampleHandler %+v\n", msg)
	return nil
}

func SampleRequestHandler(ctx context.Context, msg *brokerpb.RequestSample) error {
	m, _ := metadata.FromContext(ctx)
	log.Println("metadata", m)
	log.Printf("SampleRequestHandler %+v\n", msg)
	return nil
}

func PBStructHandler(ctx context.Context, msg *structpb.Struct) error {
	m, _ := metadata.FromContext(ctx)
	log.Println("metadata", m)
	req := zprotobuf.DecodeToMap(msg)
	// log.Printf("PBStructHandler %+v\n", msg)
	log.Printf("PBStructHandler %+v\n", req)
	return nil
}

type Request struct {
	Message  string `json:"message"`
	Count    int32  `json:"count"`
	Finished bool   `json:"finished"`
}

func JSONRequestHandler(ctx context.Context, msg *Request) error {
	m, _ := metadata.FromContext(ctx)
	log.Println("metadata", m)
	log.Printf("JSONRequestHandler %+v\n", msg)
	return nil
}

func RawDataHandler(ctx context.Context, msg *[]byte) error {
	m, _ := metadata.FromContext(ctx)
	log.Println("metadata", m)
	log.Printf("RawDataHandler %s\n", string(*msg))
	return nil
}
