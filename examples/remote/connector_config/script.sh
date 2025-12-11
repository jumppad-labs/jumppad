# Expose Kubernetes service to local machine
# k8s_ingress "k8s_web" {
#   cluster = "k8s_cluster.k3s"
#   service  = "connector-service-web"
# 
#   port {
#     local  = 9090
#     host   = 12000
#   }

#   port {
#     local  = 9091
#     host   = 12001
#   }
# }

curl localhost:9091/expose -d \
  '{
    "name":"k8s_web", 
    "local_port": 12000, 
    "remote_port": 9090, 
    "remote_server_addr": "server.k3s.k8s_cluster.local.jmpd.in:30000", 
    "service_addr": "connector-service-web:9090",
    "type": "remote"
  }'

# Expose local service to kubernetes
# local_service "local_fake" {
#   cluster = "k8s_cluster.k3s"
#   host = "172.31.92.175:19000
# 
#   port {
#     local  = 19090
#     remote = 13000
#   }

#   port {
#     local  = 19091
#     remote = 13001
#   }
#  
# }

curl localhost:9091/expose -d \
  '{
    "name":"local_fake", 
    "local_port": 19090, 
    "remote_port": 13000, 
    "remote_server_addr": "server.k3s.k8s_cluster.local.jmpd.in:30000", 
    "service_addr": "172.31.92.175:19090",
    "type": "local"
  }'