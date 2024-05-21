#!/bin/bash

load_images() {
  for file in images/*.tar.gz; do
      if [ -f "$file" ]; then
          docker load -i "$file"
      fi
  done
}

create_uuid() {
  local uuid=""
  if command -v uuidgen &>/dev/null; then
      uuid=$(uuidgen)
  else
      if [ -f /proc/sys/kernel/random/uuid ]; then
          uuid=$(cat /proc/sys/kernel/random/uuid)
      else
          uuid="fc72beb0-8bee-4f1f-955c-eceb3a287ecb"
      fi
  fi

  echo $uuid | tr '[:upper:]' '[:lower:]' > ./gateway_conf/apisix.uid
}


wait_for_service() {
  local service=$1
  local url=$2
  local retry_interval=2
  local retries=0
  local max_retry=30

  echo "Waiting for service $service to be ready..."

  while [ $retries -lt $max_retry ]; do
    if curl -k --output /dev/null --silent --head "$url"; then
      echo "Service $service is ready."
      return 0
    fi

    sleep $retry_interval
    ((retries+=1))
  done

  echo "Timeout: Service $service is not available within the specified retry limit."
  return 1
}

validate_api7_ee() {
  wait_for_service api7-ee-dashboard https://127.0.0.1:7443
}

output_listen_address() {
  if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    ips=$(ip -4 addr | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | grep -v 127.0.0.1)
  elif [[ "$OSTYPE" == "darwin"* ]]; then
    ips=$(ifconfig | grep -Eo 'inet (addr:)?([0-9]*\.){3}[0-9]*' | grep -Eo '([0-9]*\.){3}[0-9]*' | grep -v '127.0.0.1')
  fi

  for ip in $ips; do
    echo "API7-EE Listening: Dashboard(https://$ip:7443), Control Plane Address(http://$ip:7900, https://$ip:7943), Gateway(http://$ip:9080, https://$ip:9443)"
  done
  echo "If you want to access Dashboard with HTTP Endpoint(:7080), you can turn server.listen.disable to false in dashboard_conf/conf.yaml, then restart dashboard container"
}

command=${1:-start}

case $command in
    start)
        load_images
        docker compose -f docker-compose/docker-compose.yaml up -d
        # wait_for_service api7-ee-dp-manager http://127.0.0.1:7900
        # api7_registry=$(cat ./.api7_registry)
        # api7_registry_namespace=$(cat ./.api7_registry_namespace)
        # gateway_image_tag=$(cat ./.gateway_image_tag)
        # gateway_token=$(curl -k https://127.0.0.1:7443/api/gateway_groups/default/instance_token\?only_token\=true --user admin:admin -X POST)
        # if [ ! -e "./gateway_conf/apisix.uid" ]; then
        #   create_uuid
        # fi
        # docker run -d --name api7-ee-gateway-1 --network=api7-ee_api7 \
        #     -e API7_CONTROL_PLANE_ENDPOINTS='["http://dp-manager:7900"]' \
        #     -e API7_CONTROL_PLANE_TOKEN=$gateway_token \
        #     -v `pwd`/gateway_conf/config.yaml:/usr/local/apisix/conf/config.yaml \
        #     -v `pwd`/gateway_conf/apisix.uid:/usr/local/apisix/conf/apisix.uid \
        #     -p 9080:9080 \
        #     -p 9443:9443 \
        #     $api7_registry/$api7_registry_namespace/api7-ee-3-gateway:$gateway_image_tag
        # validate_api7_ee
        # echo "API7-EE is ready!"
        # output_listen_address
        ;;
    stop)
        # docker rm --force api7-ee-gateway-1
        docker compose stop
        ;;
    down)
        # docker rm --force api7-ee-gateway-1
        docker compose down
        ;;
    *)
        echo "Invalid command: $command."
        echo "  start: start the API7-EE."
        echo "  stop: stop the API7-EE."
        echo "  down: stop and remove the API7-EE."
        exit 1
        ;;
esac
