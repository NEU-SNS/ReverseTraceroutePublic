FROM ubuntu:18.04

WORKDIR /plcontroller


RUN useradd -ms /bin/bash plcontroller
RUN apt-get update && apt-get install -y \
    wget \
    build-essential
# &&  apt-get clean \
# &&  rm -rf /var/lib/apt/lists/*

RUN mkdir -p scamper && cd scamper \
    # && wget http://www.ccs.neu.edu/home/rhansen2/scamper.tar.gz \
    # && tar xzf scamper.tar.gz && cd scamper-cvs-20150901 \
     && wget http://fring2.khoury.northeastern.edu/scamper-cvs-20211212x.tar.gz \
    && tar xzf scamper-cvs-20211212x.tar.gz && cd scamper-cvs-20211212x

# For debugging scamper
# RUN mkdir -p scamper-src
# COPY scamper/scamper-cvs-20150901 /scamper-src/scamper-cvs-20150901

# WORKDIR /scamper-src/scamper-cvs-20211212x/
# RUN apt-get install --yes libc6-dev zlib1g-dev
WORKDIR /plcontroller/scamper/scamper-cvs-20211212x/
RUN apt-get install -y libssl1.0-dev
RUN ./configure
RUN ls
# RUN make LDFLAGS="-all-static" LIBS="-lssl -lcrypto -lpthread -lm -ldl " -j16
RUN make -j16
RUN  make install
RUN apt-get -y install gdb

# RUN echo $(ls -1 /plcontroller)
COPY . /plcontroller/
RUN echo $(ls -1 /plcontroller)


USER plcontroller

ENTRYPOINT ["/plcontroller/plcontroller"]
CMD ["-loglevel", "error"]

EXPOSE 4380
EXPOSE 4381
EXPOSE 55556
