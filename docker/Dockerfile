FROM ubuntu:14.04
MAINTAINER Alan Harper <alan@theexchangegroup.com.au>

RUN apt-get update
RUN apt-get -y install ca-certificates

EXPOSE 8901
ADD output/bruce /usr/bin/bruce
ENTRYPOINT /usr/bin/bruce