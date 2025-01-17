FROM golang:1.17.1-alpine3.14 as blobber_base

LABEL zchain="blobber"

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk add --update --no-cache build-base linux-headers git cmake bash perl grep

# Install Herumi's cryptography
RUN apk add gmp gmp-dev openssl-dev && \
    cd /tmp && \
    wget -O - https://github.com/herumi/mcl/archive/master.tar.gz | tar xz && \
    mv mcl* mcl && \
    make -C mcl -j $(nproc) lib/libmclbn256.so install && \
    cp mcl/lib/libmclbn256.so /usr/local/lib && \
    rm -R /tmp/mcl

RUN git clone https://github.com/herumi/bls /tmp/bls && \
    cd /tmp/bls && \
    git submodule init && \
    git submodule update && \
    make -j $(nproc) install && \
    cd - && \
    rm -R /tmp/bls
