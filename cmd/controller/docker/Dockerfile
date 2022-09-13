FROM ubuntu:latest


RUN apt-get update && apt-get install -y \
    curl \
&&  apt-get clean \
&&  rm -rf /var/lib/apt/lists/*

WORKDIR /controller
COPY . /controller

RUN useradd -ms /bin/bash controller
USER controller

ENTRYPOINT ["/controller/controller"]
CMD ["-loglevel", "error"]

EXPOSE 4382
EXPOSE 55555
EXPOSE 8080

