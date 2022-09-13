FROM ubuntu:latest

WORKDIR /vpservice
COPY . /vpservice/
RUN echo $(ls -1 /vpservice)
RUN echo $(pwd)
RUN useradd -ms /bin/bash vpservice

USER vpservice

ENTRYPOINT ["/vpservice/vpservice"]
CMD ["-loglevel", "error"]

EXPOSE 45000
EXPOSE 8080
