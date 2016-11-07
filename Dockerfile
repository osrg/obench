FROM google/debian:jessie
MAINTAINER Hitoshi Mitake <mitake.hitoshi@lab.ntt.co.jp>

RUN apt-get update && apt-get install -qq -y curl

COPY obench /bin/obench


