#!/bin/bash

cd "$(dirname $0)"
image_id=$(docker build -q ./)

docker run -v $PWD:/ganda $image_id sh -c 'cd /ganda && go build'
