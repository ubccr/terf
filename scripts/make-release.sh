#!/bin/bash

TERF_DIR='./.terf-release'
VERSION=`git describe --long --tags --dirty --always | sed -e 's/^v//'`
for os in linux darwin windows
do
    for arch in amd64 386
    do
        NAME=terf-${VERSION}-${os}-${arch}
        REL_DIR=${TERF_DIR}/${NAME}
        cd ./cmd/terf && GOOS=$os GOARCH=$arch go build -ldflags "-X main.TerfVersion=$VERSION" .
        cd ../../
        rm -Rf ${TERF_DIR}
        mkdir -p ${REL_DIR}
        cp ./cmd/terf/terf* ${REL_DIR}/
        cp ./README.rst ${REL_DIR}/
        cp ./AUTHORS.rst ${REL_DIR}/
        cp ./ChangeLog.rst ${REL_DIR}/
        cp ./LICENSE ${REL_DIR}/

        cd ${TERF_DIR} && zip -r ${NAME}.zip ${NAME}
        mv  ${NAME}.zip ../
        cd ../
        rm -Rf ${TERF_DIR}
        rm ./cmd/terf/terf*
    done
done
