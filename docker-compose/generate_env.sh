#!/bin/bash

ETCD_TAG=${ETCD_TAG:-3.5.9-debian-11-r4}
PROMETHEUS_TAG=${PROMETHEUS_TAG:-2.44.0-debian-11-r7}
POSTGRESQL_TAG=${POSTGRESQL_TAG:-15.3.0-debian-11-r0}
API7_REGISTRY=${API7_REGISTRY:-hkccr.ccs.tencentyun.com}
API7_REGISTRY_NAMESPACE=${API7_REGISTRY_NAMESPACE:-api7}
API7_DP_MANAGER_TAG=${API7_DP_MANAGER_TAG:-dev}
API7_GATEWAY_TAG=${API7_GATEWAY_TAG:-dev}
API7_DASHBOARD_TAG=${API7_DASHBOARD_TAG:-dev}
API7_NETWORK=${API7_NETWORK:-kind}

echo "ETCD_TAG=${ETCD_TAG}" > ./docker-compose/.env
echo "PROMETHEUS_TAG=${PROMETHEUS_TAG}" >> ./docker-compose/.env
echo "POSTGRESQL_TAG=${POSTGRESQL_TAG}" >> ./docker-compose/.env
echo "API7_REGISTRY=${API7_REGISTRY}" >> ./docker-compose/.env
echo "API7_REGISTRY_NAMESPACE=${API7_REGISTRY_NAMESPACE}" >> ./docker-compose/.env
echo "API7_DP_MANAGER_TAG=${API7_DP_MANAGER_TAG}" >> ./docker-compose/.env
echo "API7_DASHBOARD_TAG=${API7_DASHBOARD_TAG}" >> ./docker-compose/.env

echo -n "${API7_REGISTRY}" > ./docker-compose/.api7_registry
echo -n "${API7_REGISTRY_NAMESPACE}" > ./docker-compose/.api7_registry_namespace
echo -n "${API7_GATEWAY_TAG}" > ./docker-compose/.gateway_image_tag
