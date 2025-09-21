# Docker Compose Ansible Windows

## Just start container without running any commands

```bash
docker-compose -f deployments/docker-compose/ansible-windows/docker-compose.keepalive.yml up
```

### Then exec into the container

```bash
docker exec -it ansible-windows-test C:\cygwin64\bin\bash.exe -l
```

### From inside the container, run ansible commands

```bash
cd /cygdrive/c/src/
ansible-playbook playbooks/setup.yml -i inventory/localhost_windows.yml -l windows_host -v
```

## Run ansible directly without entering the container

```bash
docker-compose -f deployments/docker-compose/ansible-windows/docker-compose.yml up
```
