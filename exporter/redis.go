package exporter

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	r_kvstore "github.com/aliyun/alibaba-cloud-sdk-go/services/r-kvstore"
	"github.com/prometheus/client_golang/prometheus"
)

var sleep = false

func wakeup() {
	lock := sync.RWMutex{}
	lock.Lock()
	time.Sleep(time.Minute * time.Duration(1))
	sleep = false
	lock.Unlock()
}

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
	var newMetric []string
	if extraMetric != "" {
		newMetric = append(metric, strings.Split(extraMetric, ",")...)
	} else {
		newMetric = metric
	}

	it := make(map[string]string)
	for _, v := range r.instances {
		if v.instanceType == SHARDING {
			it["Sharding"] = ""
		} else if v.instanceType == STANDARD {
			it["Standard"] = ""
		} else {
			it["Splitrw"] = ""
		}
	}

	var m []string
	for k := range it {
		for _, v := range newMetric {
			m = append(m, fmt.Sprintf("%s%s", k, v))
		}
	}
	r.metricMetas = m
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

func (r *RedisExporter) InitGauge() {
	r.NewClient()
	r.GetInstance()
	r.GetMetricMeta()
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			r.GetInstance()
			r.GetMetricMeta()
		}
	}()
	r.metrics = map[string]*prometheus.GaugeVec{}
	for _, m := range r.metricMetas {
		name := ""
		if strings.HasPrefix(m, "Standard") {
			name = strings.TrimPrefix(m, "Standard")
		} else if strings.HasPrefix(m, "Sharding") {
			name = strings.TrimPrefix(m, "Sharding")
		} else {
			name = strings.TrimPrefix(m, "Splitrw")
		}
		r.metrics[m] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "aliyun_redis",
			Name:      strings.ToLower(name),
		}, []string{"instance_id", "instance_name", "instance_type", "node_id"})

	}
}

func (r *RedisExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, v := range r.metrics {
		v.Describe(ch)
	}
}

func (r *RedisExporter) Collect(ch chan<- prometheus.Metric) {
	if !sleep {
		sleep = true
		go wakeup()
		for _, m := range r.metricMetas {
			r.GetMetric(m)
			for _, d := range r.DataPoints {
				instanceName, instanceType := r.GetInstanceNameTypeById(d.InstanceId)
				if instanceType == STANDARD {
					r.metrics[m].With(
						prometheus.Labels{
							"instance_id":   d.InstanceId,
							"instance_name": instanceName,
							"instance_type": instanceType,
							"node_id":       d.InstanceId,
						}).Set(d.Average)
				} else {
					r.metrics[m].With(
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
