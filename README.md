### docker build
``` docker build . -t <image>:<tag> ```

### or pull 
``` docker pull riet/aliyun-redis-exporter ```

### run
with default metrics
```
docker run \ 
  -d \ 
  --name aliyun-slb-exporter \
  -e ACCESS_KEY_ID=<aliyun ak> \
  -e ACCESS_KEY_SECRET=<aliyun ak sk> \
  -e REGION_ID=<region id> \
  -p 10003:10003 \
  riet/aliyun-redis-exporter 
```

with extra metrics
```
docker run \ 
  -d \ 
  --name aliyun-slb-exporter \
  -e ACCESS_KEY_ID=<aliyun ak> \
  -e ACCESS_KEY_SECRET=<aliyun ak sk> \
  -e REGION_ID=<region id> \
  -e EXTRA_METRIC=metric1,metric2,metric3
  -p 10003:10003 \
  riet/aliyun-redis-exporter 
```
extar metric name list: https://help.aliyun.com/document_detail/28619.html?spm=a2c4g.11186623.2.12.719e1a35jqlolt#title-mwp-ec2-2c6  
the metric name should remove prefix of "Sharding" "Standard" "Splitrw"

visit http://localhost:10003/metrics