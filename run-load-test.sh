#!/usr/bin/env bash

set -Eeuo pipefail

########################################
# validate gcloud credential
########################################
GCP_ENV="develop"
GCP_PROJECT_ID=$(gcloud projects list --filter="name=${GCP_ENV}" --format="value(projectId)")

test -z "${GCP_PROJECT_ID}" && echo 'ERR! check and authorize your glcoud config first' && exit 1;

########################################
# do something before exiting script
########################################
trap 'cleanup' SIGINT SIGTERM ERR EXIT

cleanup() {
  trap - SIGINT SIGTERM ERR EXIT
  ########################################
  # clearup backgroud processes
  ########################################
  set -x
  echo "clearup before exiting"
  test -n "`docker ps -a --filter NAME=^${CONTAINER_NAME}$ --format {{.ID}}`" && docker rm -f ${CONTAINER_NAME}
}

########################################
# GKE port-forward at background
########################################
GKE_CLUSTER_NAME=`gcloud container clusters list --filter='NAME~cp-develop-gke-backend' --format='(NAME)' --project=${GCP_PROJECT_ID} | tail -n1`
GKE_CLUSTER_LOCATION=`gcloud container clusters list --format='value(location)' --project=${GCP_PROJECT_ID}`

CONTAINER_NAME="gke_port_forward"
CONTAINER_PORT="4399"
POD_NS="backend"
POD_KEYWORD="cp-${GCP_ENV}-polkadot-subscan-api"
POD_NAME=`kubectl -n "${POD_NS}" get pods | grep "${POD_KEYWORD}" | awk '{print $1}'`

docker run -d --rm --name ${CONTAINER_NAME} --entrypoint=bash \
  -v ${HOME}/.config/gcloud:/root/.config/gcloud \
  -e GCP_PROJECT_ID=${GCP_PROJECT_ID} \
  -e GKE_CLUSTER_NAME=${GKE_CLUSTER_NAME} \
  -e GKE_CLUSTER_LOCATION=${GKE_CLUSTER_LOCATION} \
  -e CONTAINER_PORT=${CONTAINER_PORT} \
  -e POD_NS=${POD_NS} \
  -e POD_NAME=${POD_NAME} \
  gcr.io/google.com/cloudsdktool/cloud-sdk:latest -c '
    gcloud container clusters get-credentials ${GKE_CLUSTER_NAME} --zone ${GKE_CLUSTER_LOCATION} --project ${GCP_PROJECT_ID}

    echo "GKE prot-forward start: ${POD_NAME}:${CONTAINER_PORT}"
    kubectl port-forward pods/${POD_NAME} ${CONTAINER_PORT} --address 0.0.0.0 --namespace ${POD_NS} &

    while [[ ! -e "/workspace/TEST_STOPPED" ]]; do echo wait for TEST_STOPPED; sleep 10s; done;
'

CONTAINER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ${CONTAINER_NAME}`

echo "WAIT! 10 sec for ${CONTAINER_NAME} ready" && sleep 10s

########################################
# load test with k6
########################################
docker run -it --rm --name load-test \
  -v `pwd`:/app \
  -w /app \
  loadimpact/k6 run test/load-test/load-test.js \
  -e HOST=http://${CONTAINER_IP}:${CONTAINER_PORT} \
  -e HARD_MODE=1
