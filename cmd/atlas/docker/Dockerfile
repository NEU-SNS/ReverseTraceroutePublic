FROM ubuntu:latest

# Install python3
RUN apt-get -y update
RUN apt-get install python3 -y
RUN apt-get install ca-certificates -y
RUN apt-get install sudo -y
RUN apt-get install iputils-ping -y
RUN apt-get install net-tools -y
RUN apt-get install telnet -y
RUN apt-get install openssh-client -y
RUN apt-get -y update
RUN apt-get install python3-pip -y
RUN pip3 install mysql-connector 

WORKDIR /atlas
COPY . /atlas


RUN useradd -ms /bin/bash atlas && usermod -aG sudo atlas

# Move the rankingserviice code in the right place
ARG REVTR_DIR=/home/atlas/go/src/github.com/NEU-SNS/ReverseTraceroute/
RUN mkdir -p $REVTR_DIR
RUN mv rankingservice $REVTR_DIR 
RUN chown -R atlas $REVTR_DIR
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

USER atlas

ENTRYPOINT ["/atlas/atlas.sh"]
CMD ["-loglevel", "error"]

EXPOSE 55000
EXPOSE 8080
