yard up \
    --type k3s \
    --name cloud \
    --consul-port 18500 \
    --dashboard-port 18443 \
    --network $(PROJECT)_wan \
    --network-ip "192.169.7.100" \
    --consul-values $(CLOUD_PATH)/consul-values.yaml \
	--push-image nicholasjackson/fake-service:v0.7.7 \
	--push-image nicholasjackson/fake-service:vm-v0.7.7 \
	--push-image envoyproxy/envoy:v1.10.0 \
	--push-image kubernetesui/dashboard:v2.0.0-beta4

yard expose \
    --name cloud \
    --bind-ip none \
	--network $(PROJECT)_wan \
	--network-ip 192.169.7.130 \
	--service-name svc/consul-consul-server \
	--port 8600:8600 \
	--port 8500:8500 \
	--port 8302:8302 \
	--port 8301:8301 \
	--port 8300:8300

yard expose --name multi-cloud --bind-ip none \
	--network $(PROJECT)_wan \
	--network-ip 192.169.7.240 \
	--service-name svc/consul-consul-mesh-gateway \
	--port 443:443