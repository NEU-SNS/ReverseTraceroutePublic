FROM ubuntu:latest

RUN apt-get -y update
RUN apt-get install openssh-client -y
RUN apt-get install net-tools -y
# Install python for pyasn
RUN apt-get install -y python3
RUN apt-get -y update
RUN apt-get install -y python3-pip 
RUN apt-get install -y wget 
RUN apt-get install -qq -y ca-certificates &&  apt-get clean \
&&  rm -rf /var/lib/apt/lists/*

RUN pip3 install pyasn


WORKDIR /revtr
COPY . /revtr



ENTRYPOINT ["/revtr/revtr.sh"]
CMD ["-loglevel", "error"]

EXPOSE 8080
EXPOSE 8181
EXPOSE 9999
EXPOSE 45454