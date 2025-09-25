FROM mcr.microsoft.com/windows/servercore:ltsc2025

SHELL ["powershell", "-Command", "$ErrorActionPreference = 'Stop';"]

# Install chocolatey
RUN Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# Install Cygwin
RUN choco install -y cygwin --params \"/InstallDir:C:\cygwin64 /NoAdmin /NoStartMenu\"

RUN choco install -y cyg-get
RUN cyg-get python39 python39-pip python39-cryptography openssh git make gcc-core gcc-g++ libffi-devel openssl-devel

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

ENTRYPOINT ["C:\\cygwin64\\bin\\bash.exe", "-l"]
CMD ["-c", "while true; do sleep 30; done;"]