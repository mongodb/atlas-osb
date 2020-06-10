FROM ubuntu:bionic
ENV TILE_GENERATOR_VERSION=12.0.8 \
    YQ_VERSION=3.3.0 \
    PIVNET_CLI_VERSION=0.0.55 \
    CREDHUB_CLI_VERSION=2.2.0 \
    OM_VERSION=4.0.1 \
    BOSH_CLI_VERSION=5.4.0 \
    BBL_VERSION=7.1.0 \
    DUMB_INIT_VERSION=1.2.2 \
    GOLANG_VERSION=1.13.3 \
    CF_CLI_VERSION=6.51.0 \
    PATH=/go/bin:/usr/local/go/bin:/root/google-cloud-sdk/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    GOPATH=/go
RUN set -xe && \
    echo '#!/bin/sh' > /usr/sbin/policy-rc.d && \
    echo 'exit 101' >> /usr/sbin/policy-rc.d && \
    chmod +x /usr/sbin/policy-rc.d && \
    dpkg-divert --local --rename --add /sbin/initctl && \
    cp -a /usr/sbin/policy-rc.d /sbin/initctl && \
    sed -i 's/^exit.*/exit 0/' /sbin/initctl && \
    echo 'force-unsafe-io' > /etc/dpkg/dpkg.cfg.d/docker-apt-speedup && \ 
    echo 'DPkg::Post-Invoke { "rm -f /var/cache/apt/archives/*.deb /var/cache/apt/archives/partial/*.deb /var/cache/apt/*.bin || true"; };' > /etc/apt/apt.conf.d/docker-clean  && \
    echo 'APT::Update::Post-Invoke { "rm -f /var/cache/apt/archives/*.deb /var/cache/apt/archives/partial/*.deb /var/cache/apt/*.bin || true"; };' >> /etc/apt/apt.conf.d/docker-clean  && \
    echo 'Dir::Cache::pkgcache ""; Dir::Cache::srcpkgcache "";' >> /etc/apt/apt.conf.d/docker-clean && \
    echo 'Acquire::Languages "none";' > /etc/apt/apt.conf.d/docker-no-languages && \
    echo 'Acquire::GzipIndexes "true"; Acquire::CompressionTypes::Order:: "gz";' > /etc/apt/apt.conf.d/docker-gzip-indexes && \
    echo 'Apt::AutoRemove::SuggestsImportant "false";' > /etc/apt/apt.conf.d/docker-autoremove-suggests && \ 
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /run/systemd && echo 'docker' > /run/systemd/container && \
    rm -f /etc/localtime && \
    ln -sf /usr/share/zoneinfo/Etc/UTC /etc/localtime
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive && \
    apt-get install -y --no-install-recommends build-essential git ruby-dev openssh-client unzip wget curl rsync vim-nox python openjdk-8-jdk jq && \
    apt-get autoremove && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
RUN wget https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64 -O yq && \
    mv yq /usr/local/bin/yq && \
    chmod +x /usr/local/bin/yq && \
    wget https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-${BOSH_CLI_VERSION}-linux-amd64 -O bosh && \
    mv bosh /usr/local/bin/bosh && \
    chmod +x /usr/local/bin/bosh && \
    wget https://github.com/pivotal-cf/om/releases/download/${OM_VERSION}/om-linux-${OM_VERSION} -O om-linux && \
    mv om-linux /usr/local/bin/om && \
    chmod +x /usr/local/bin/om && \
    wget https://github.com/cloudfoundry-incubator/credhub-cli/releases/download/${CREDHUB_CLI_VERSION}/credhub-linux-${CREDHUB_CLI_VERSION}.tgz -O credhub.tgz && \ 
    tar xzf ./credhub.tgz && \
    mv credhub /usr/local/bin/credhub && \
    chmod +x /usr/local/bin/credhub && \
    rm -f ./credhub.tgz && \
    wget https://s3-us-west-1.amazonaws.com/cf-cli-releases/releases/v${CF_CLI_VERSION}/cf-cli_${CF_CLI_VERSION}_linux_x86-64.tgz -O cf.tgz && \
    tar xzf ./cf.tgz cf && \
    mv cf /usr/local/bin/cf && \
    chmod +x /usr/local/bin/cf && \
    rm -f ./cf.tgz && \
    wget https://github.com/pivotal-cf/pivnet-cli/releases/download/v${PIVNET_CLI_VERSION}/pivnet-linux-amd64-${PIVNET_CLI_VERSION} -O pivnet && \
    mv pivnet /usr/local/bin/pivnet && \
    chmod +x /usr/local/bin/pivnet && \
    wget https://github.com/cf-platform-eng/tile-generator/releases/download/v${TILE_GENERATOR_VERSION}/pcf_linux-64bit -O pcf && \
    mv pcf /usr/local/bin/pcf && \
    chmod +x /usr/local/bin/pcf && \
    wget https://github.com/cf-platform-eng/tile-generator/releases/download/v${TILE_GENERATOR_VERSION}/tile_linux-64bit -O tile && \
    mv tile /usr/local/bin/tile && \
    chmod +x /usr/local/bin/tile && \
    wget https://storage.googleapis.com/golang/go${GOLANG_VERSION}.linux-amd64.tar.gz -P /tmp && \
    tar xzf /tmp/go${GOLANG_VERSION}.linux-amd64.tar.gz -C /usr/local && \
    mkdir $GOPATH && \
    go get -u github.com/onsi/ginkgo/... && \
    wget https://github.com/cloudfoundry/bosh-bootloader/releases/download/v${BBL_VERSION}/bbl-v${BBL_VERSION}_linux_x86-64 -O bbl && \
    mv bbl /usr/local/bin/bbl && \
    chmod +x /usr/local/bin/bbl 
    # wget https://github.com/Yelp/dumb-init/releases/download/v${DUMB_INIT_VERSION}/dumb-init_${DUMB_INIT_VERSION}_amd64 -O dumb-init && \
    # mv dumb-init /usr/local/bin/dumb-init && \
    # chmod +x /usr/local/bin/dumb-init && \
CMD ["/bin/bash"]
