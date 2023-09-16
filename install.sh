#!/bin/sh
echo 'Installing docker log driver loki'
docker plugin install grafana/loki-docker-driver --alias loki --grant-all-permissions
