package exporter

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	r_kvstore "github.com/aliyun/alibaba-cloud-sdk-go/services/r-kvstore"
	"github.com/prometheus/client_golang/prometheus"
)

func (r *RedisExporter) NewClient() {
	client, err := cms.NewClientWithAccessKey(regionId, accessKeyId, accessKeySecret)
	if err != nil {
		panic(err)
	}
	r.client = client
}

func (r *RedisExporter) GetInstance() {
	client, err := r_kvstore.NewClientWithAccessKey(regionId, accessKeyId, accessKeySecret)
	if err != nil {
		panic(err)
	}

	request := r_kvstore.CreateDescribeInstancesRequest()
	request.PageSize = requests.NewInteger(50)
	response, err := client.DescribeInstances(request)
	if err != nil {
		panic(err)
	}
	var instances []RedisInstance
	for _, v := range response.Instances.KVStoreInstance {
		instances = append(instances, RedisInstance{instanceId: v.InstanceId, instanceName: v.InstanceName, instanceType: v.ArchitectureType})
	}
	r.instances = instances
}

func (r *RedisExporter) GetMetricMeta() {
	request := cms.CreateDescribeMetricMetaListRequest()
	request.Namespace = PROJECT
	request.PageSize = requests.NewInteger(600)
	response, err := r.client.DescribeMetricMetaList(request)
	if err != nil {
		panic(err)
	}
	for _, v := range response.Resources.Resource {
		if strings.HasPrefix(v.MetricName, "Shard") {
			r.shardingMetricMetas = append(r.shardingMetricMetas, v.MetricName)
		}
		if strings.HasPrefix(v.MetricName, "Stand") {
			r.standardMetricMetas = append(r.standardMetricMetas, v.MetricName)
		}
		if strings.HasPrefix(v.MetricName, "Split") {
			r.splitMetricMetas = append(r.splitMetricMetas, v.MetricName)
		}
	}
}

func (r *RedisExporter) GetMetric(metricName string) {
	var dimensions []map[string]string
	for _, v := range r.instances {
		d := map[string]string{"instanceId": v.instanceId}
		dimensions = append(dimensions, d)
	}
	dimension, err := json.Marshal(dimensions)
	if err != nil {
		log.Println(err)
	}

	request := cms.CreateDescribeMetricLastRequest()
	request.Namespace = PROJECT
	request.MetricName = metricName
	request.Dimensions = string(dimension)
	request.Period = "120"
	response, err := r.client.DescribeMetricLast(request)
	if err != nil {
		log.Println(err)
	}
	err = json.Unmarshal([]byte(response.Datapoints), &r.DataPoints)
	if err != nil {
		log.Println(err)
	}
}

func (r *RedisExporter) GetInstanceNameTypeById(instanceId string) (string, string) {
	for _, v := range r.instances {
		if v.instanceId == instanceId {
			if v.instanceName != "" {
				return v.instanceName, v.instanceType
			}
			return v.instanceId, v.instanceType
		}
	}
	return "", ""
}

func (r *RedisExporter) GetMetricMetaByInstance(i RedisInstance) []string {
	var metricMetas []string
	if i.instanceType == SHARDING {
		metricMetas = r.shardingMetricMetas
	} else if i.instanceType == STANDARD {
		metricMetas = r.standardMetricMetas
	} else if i.instanceType == SPLITRW {
		metricMetas = r.splitMetricMetas
	}
	return metricMetas
}

func (r *RedisExporter) InitGauge() {
	r.NewClient()
	r.GetInstance()
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			r.GetInstance()
		}
	}()
	r.GetMetricMeta()
	r.metrics = map[string]*prometheus.GaugeVec{}
	for _, v := range r.instances {
		metricMetas := r.GetMetricMetaByInstance(v)
		for _, m := range metricMetas {
			name := ""
			if strings.HasPrefix(m, "Standard") {
				name = strings.TrimPrefix(m, "Standard")
				r.metrics[v.instanceId+"_"+m] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
					Namespace: "aliyun_redis",
					Name:      strings.ToLower(name),
				}, []string{"instance_id", "instance_name", "instance_type"})
			} else {
				if strings.HasPrefix(m, "Sharding") {
					name = strings.TrimPrefix(m, "Sharding")
				} else {
					name = strings.TrimPrefix(m, "Splitrw")
				}
				r.metrics[v.instanceId+"_"+m] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
					Namespace: "aliyun_redis",
					Name:      strings.ToLower(name),
				}, []string{"instance_id", "instance_name", "instance_type", "node_id"})
			}

		}
	}
}

func (r RedisExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, v := range r.metrics {
		v.Describe(ch)
	}
}

func (r RedisExporter) Collect(ch chan<- prometheus.Metric) {
	for _, v := range r.instances {
		metricMetas := r.GetMetricMetaByInstance(v)
		for _, m := range metricMetas {
			r.GetMetric(m)
			for _, d := range r.DataPoints {
				instanceName, instanceType := r.GetInstanceNameTypeById(d.InstanceId)
				if instanceType == STANDARD {
					r.metrics[v.instanceId+"_"+m].With(
						prometheus.Labels{
							"instance_id":   d.InstanceId,
							"instance_name": instanceName,
							"instance_type": instanceType,
						}).Set(d.Average)
				} else {
					r.metrics[v.instanceId+"_"+m].With(
						prometheus.Labels{
							"instance_id":   d.InstanceId,
							"instance_name": instanceName,
							"instance_type": instanceType,
							"node_id":       d.NodeId,
						}).Set(d.Average)
				}

			}
			time.Sleep(34 * time.Millisecond)
		}
	}
	for _, m := range r.metrics {
		m.Collect(ch)
	}
}
