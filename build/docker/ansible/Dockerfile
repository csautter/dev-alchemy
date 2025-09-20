FROM ubuntu:24.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    software-properties-common

RUN add-apt-repository --yes --update ppa:ansible/ansible

RUN apt install -y ansible

RUN ansible --version