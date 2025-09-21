FROM mcr.microsoft.com/windows/servercore:ltsc2025

SHELL ["powershell", "-Command", "$ErrorActionPreference = 'Stop';"]

# Install Cygwin
ADD https://cygwin.com/setup-x86_64.exe C:\\cygwin\\setup-x86_64.exe

# Install Cygwin with base packages + python + pip + openssh
RUN C:\\cygwin\\setup-x86_64.exe -q -n -N -d -R C:\\cygwin64 \
    -s https://ftp.fau.de/cygwin/ \
    -P python39,python39-pip,python39-cryptography,openssh,git,make,gcc-core,gcc-g++,libffi-devel,openssl-devel

# Install Ansible inside Cygwin
RUN C:\\cygwin64\\bin\\python3.9.exe -m pip install ansible
RUN C:\\cygwin64\\bin\\python3.9.exe -m pip install pywinrm

# enable winrm
RUN Set-Item -Path WSMan:\localhost\Service\AllowUnencrypted -Value $true; \
    Set-Item -Path WSMan:\localhost\Service\Auth\Basic -Value $true; \
    Enable-PSRemoting -Force

# hardcoded passwords are bad, but this is just for local testing
RUN net user ansible 'Secret123!@#' /add; \
    net localgroup Administrators ansible /add

# Default shell inside the container
ENTRYPOINT ["C:\\cygwin64\\bin\\bash.exe", "-l"]
CMD ["-c", "while true; do sleep 30; done;"]