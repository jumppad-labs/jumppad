---
title: Single Container Example
author: Nic Jackson
slug: container
browser_windows: http://consul-http.ingress.shipyard.run:8500
env:
  - SOMETHING=else
shipyard_version: ">= v0.0.37"
---

# Single Container

This blueprint shows how you can create a single container with Shipyard

```shell
curl http://consul-http.ingress.shipyard.run:8500
```