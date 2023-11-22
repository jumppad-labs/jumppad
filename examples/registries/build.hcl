
resource "certificate_leaf" "registry" {
  ca_key  = "${jumppad()}/certs/root.key"
  ca_cert = "${jumppad()}/certs/root.cert"

  ip_addresses = ["127.0.0.1", variable.auth_ip_address, variable.noauth_ip_address]

  dns_names = [
    "localhost",
    "auth-registry.demo.gs",
    "noauth-registry.demo.gs", // have to set an external dns name as the registry resolves docker dns to localhost
    "noauth.container.jumppad.dev",
    "auth.container.jumppad.dev",
  ]

  output = data("certs")
}

resource "container" "noauth" {
  image {
    name = "registry:2"
  }

  network {
    id         = resource.network.cloud.id
    ip_address = variable.noauth_ip_address
  }

  port {
    local = 443
    host  = 5000
  }

  environment = {
    DEBUG                         = "true"
    REGISTRY_HTTP_ADDR            = "0.0.0.0:443"
    REGISTRY_HTTP_TLS_CERTIFICATE = "/certs/registry-leaf.cert"
    REGISTRY_HTTP_TLS_KEY         = "/certs/registry-leaf.key"
  }

  volume {
    source      = data("certs")
    destination = "/certs"
  }
}

resource "container" "auth" {
  image {
    name = "registry:2"
  }

  network {
    id         = resource.network.cloud.id
    ip_address = variable.auth_ip_address
  }

  port {
    local = 443
    host  = 5001
  }

  environment = {
    DEBUG                         = "true"
    REGISTRY_HTTP_ADDR            = "0.0.0.0:443"
    REGISTRY_AUTH                 = "htpasswd"
    REGISTRY_AUTH_HTPASSWD_REALM  = "Registry Realm"
    REGISTRY_AUTH_HTPASSWD_PATH   = "/etc/auth/htpasswd"
    REGISTRY_HTTP_TLS_CERTIFICATE = "/certs/registry-leaf.cert"
    REGISTRY_HTTP_TLS_KEY         = "/certs/registry-leaf.key"
  }

  volume {
    source      = "./files/htpasswd"
    destination = "/etc/auth/htpasswd"
  }

  volume {
    source      = data("certs")
    destination = "/certs"
  }
}

resource "container" "insecure" {
  image {
    name = "registry:2"
  }

  network {
    id         = resource.network.cloud.id
    ip_address = variable.insecure_ip_address
  }

  port {
    local = 5003
    host  = 5003
  }

  environment = {
    DEBUG              = "true"
    REGISTRY_HTTP_ADDR = "0.0.0.0:5003"
  }
}

resource "build" "app" {
  container {
    dockerfile = "./Docker/Dockerfile"
    context    = "./src"
    ignore     = ["**/.terraform"]
  }

  // push to the unauthenticated registry
  registry {
    name = "${resource.container.noauth.container_name}:5000/mine:v0.1.0"
  }

  // push to the authenticated registry
  registry {
    name     = "${resource.container.auth.container_name}:5001/mine:v0.1.0"
    username = "admin"
    password = "password"
  }

  // push to the insecure registry
  registry {
    name = "${resource.container.insecure.container_name}:5003/mine:v0.1.0"
  }
}

# Define a custom registry that will be added to the image cache
resource "container_registry" "noauth" {
  hostname = "noauth-registry.demo.gs" // cache can not resolve local jumppad.dev dns for some reason, 
  // using external dns mapped to the local ip address
}

resource "container_registry" "auth" {
  hostname = "auth-registry.demo.gs"
  auth {
    username = "admin"
    password = "password"
  }
}
