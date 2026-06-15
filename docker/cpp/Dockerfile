FROM ubuntu:24.04

RUN apt-get update && \
    apt-get install -y g++ && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
RUN chmod 777 /workspace

CMD ["g++", "--version"]

# Removing the /var/lib/apt/lists to reduce the image size