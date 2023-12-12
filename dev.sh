#!/bin/bash
cd urtty-fe
DOCKER_BUILDKIT=0 docker build -t urtty-builder .
cd ..
container_id=$(docker create urtty-builder)
rm -rf web
docker cp $container_id:/webui/build ./web
