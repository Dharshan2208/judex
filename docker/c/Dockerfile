FROM ubuntu:24.04

RUN apt-get update && \
    apt-get install -y gcc && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

CMD ["gcc", "--version"]

# Removing the /var/lib/apt/lists to reduce the image size