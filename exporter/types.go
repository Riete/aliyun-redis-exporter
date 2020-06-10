package exporter

import (
	"os"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	accessKeyId     = os.Getenv("ACCESS_KEY_ID")
	accessKeySecret = os.Getenv("ACCESS_KEY_SECRET")
	regionId        = os.Getenv("REGION_ID")
)

const (
	PROJECT  string = "acs_kvstore"
	SHARDING string = "cluster"
	STANDARD string = "standard"
	SPLITRW  string = "SplitRW"
)

type RedisInstance struct {
	instanceId   string
	instanceName string
	instanceType string
}

type RedisExporter struct {
	client              *cms.Client
	metrics             map[string]*prometheus.GaugeVec
	instances           []RedisInstance
	shardingMetricMetas []string
	splitMetricMetas    []string
	standardMetricMetas []string
	DataPoints          []struct {
		InstanceId string  `json:"instanceId"`
		Average    float64 `json:"Average"`
		NodeId     string  `json:"nodeId,omitempty"`
	}
}
