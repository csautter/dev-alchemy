FROM mcr.microsoft.com/windows/servercore:ltsc2025

SHELL ["powershell", "-Command", "$ErrorActionPreference = 'Stop';"]

# Install chocolatey
RUN Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# Install Cygwin
RUN choco install -y cygwin --params \"/InstallDir:C:\cygwin64 /NoAdmin /NoStartMenu\"

RUN choco install -y cyg-get
RUN cyg-get python39 python39-pip python39-cryptography openssh git make gcc-core gcc-g++ libffi-devel openssl-devel sshpass

# Install Ansible inside Cygwin
RUN C:\\cygwin64\\bin\\python3.9.exe -m pip install ansible

# Enable and configure Windows native OpenSSH server
RUN "Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0"
RUN "Start-Service sshd; Set-Service -Name sshd -StartupType 'Automatic'"

# hardcoded passwords are bad, but this is just for local testing
RUN net user ansible 'Secret123!@#' /add; \
    net localgroup Administrators ansible /add

# Set default shell for OpenSSH to PowerShell, otherwise Ansible will fail with the default cmd.exe
RUN New-ItemProperty -Path 'HKLM:\SOFTWARE\OpenSSH' -Name 'DefaultShell' -Value 'C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe' -PropertyType String -Force

ENTRYPOINT ["C:\\cygwin64\\bin\\bash.exe", "-l"]
CMD ["-c", "while true; do sleep 30; done;"]