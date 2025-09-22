# Docker Compose Ansible Windows

## Switch to Windows containers

Make sure you are running Windows containers in Docker Desktop.
Link: https://learn.microsoft.com/en-us/virtualization/windowscontainers/quick-start/set-up-environment?tabs=dockerce#windows-10-and-windows-11-2

## Just start container without running any commands

```bash
# use ansible with winrm protocol
docker-compose -f deployments/docker-compose/ansible-windows/docker-compose.keepalive.yml up ansible-windows-winrm
# use ansible with ssh protocol
docker-compose -f deployments/docker-compose/ansible-windows/docker-compose.keepalive.yml up ansible-windows-ssh
```

### Then exec into the container

```bash
# use ansible with winrm protocol
docker exec -it ansible-windows-winrm C:\cygwin64\bin\bash.exe -l
# use ansible with ssh protocol
docker exec -it ansible-windows-ssh C:\cygwin64\bin\bash.exe -l
```

### From inside the container, run ansible commands

```bash
cd /cygdrive/c/src/
# use ansible with winrm protocol
ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_winrm.yml -l windows_host -vvv
# use ansible with ssh protocol
ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_ssh.yml -l windows_host --ask-pass -vvv
```

## Run ansible directly without entering the container

```bash
docker-compose -f deployments/docker-compose/ansible-windows/docker-compose.yml up
```
