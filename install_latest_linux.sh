#!/bin/bash

export LATEST=$(curl -s -L https://api.github.com/repos/cloudshare/docker-machine-driver-cloudshare/releases/latest | grep tag_name | grep -E "[.0-9]+" -o) && \
    curl -s -L https://github.com/cloudshare/docker-machine-driver-cloudshare/releases/download/${LATEST}/docker-machine-driver-cloudshare_amd64-linux.tar.gz -o /tmp/docker-machine-driver-cloudshare.tar.gz && \
    cd /tmp && tar xf /tmp/docker-machine-driver-cloudshare.tar.gz && \
    mv /tmp/docker-machine-driver-cloudshare /usr/local/bin/ && \
    chmod +x /usr/local/bin/docker-machine-driver-cloudshare
