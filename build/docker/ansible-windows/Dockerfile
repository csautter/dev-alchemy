# python:3.13-windowsservercore-ltsc2025
FROM python:3.12-windowsservercore-ltsc2025
#FROM mcr.microsoft.com/windows-cssc/python:3.13-windows-ltsc2019

SHELL ["powershell", "-Command", "$ErrorActionPreference = 'Stop';"]

# Install Ansible
RUN python -m pip install ansible-core pywinrm

# Verify Ansible installation
RUN ansible --version

ENTRYPOINT [ "ansible-playbook" ]
CMD [ "--help" ]