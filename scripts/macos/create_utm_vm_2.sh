#!/bin/bash

# utmctl_create_macos_vm.sh
# This script automates the creation of a macOS virtual machine using utmctl.
# It requires the UTM application to be installed. It can now automatically
# download the latest macOS IPSW file if one is not provided.

# --- Configuration Variables ---
VM_NAME="macOS_VM_$(date +%Y%m%d%H%M%S)" # Default VM name with timestamp
RAM_GB=8                                  # Default RAM in GB
CPU_CORES=4                               # Default CPU cores
DISK_SIZE_GB=100                          # Default disk size in GB
IPSW_PATH=""                              # Optional: Set the path to your macOS IPSW file here.
                                          # If empty, the script will attempt to download the latest.
                                          # Example: IPSW_PATH="/Users/youruser/Downloads/UniversalMac_14.5_23F79_Restore.ipsw"
DOWNLOAD_DIR="${HOME}/Downloads"          # Directory to save downloaded IPSW file

# --- Functions ---

# Function to display script usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "Automates the creation of a macOS VM using utmctl."
    echo ""
    echo "Options:"
    echo "  -n, --name <name>      Set the VM name (default: $VM_NAME)"
    echo "  -r, --ram <GB>         Set RAM in GB (default: $RAM_GB)"
    echo "  -c, --cpu <cores>      Set CPU cores (default: $CPU_CORES)"
    echo "  -d, --disk <GB>        Set disk size in GB (default: $DISK_SIZE_GB)"
    echo "  -i, --ipsw <path>      Optional: Path to the macOS IPSW restore image."
    echo "                         If not provided, the script will attempt to download the latest."
    echo "  -s, --start            Start the VM after creation"
    echo "  -h, --help             Display this help message"
    echo ""
    echo "Example:"
    echo "  $0 -n MySonomaVM -r 16 -c 6 -d 150 -i /path/to/UniversalMac_14.5_Restore.ipsw -s"
    echo "  $0 -s # To create and start with default settings and auto-downloaded IPSW"
    exit 1
}

# Function to check if utmctl is available
check_utmctl() {
    if ! command -v utmctl &> /dev/null; then
        echo "Error: utmctl command not found."
        echo "Please ensure UTM is installed and utmctl is in your PATH."
        echo "You might need to create a symlink: sudo ln -s /Applications/UTM.app/Contents/MacOS/utmctl /usr/local/bin/utmctl"
        exit 1
    fi
}

# Function to download the latest macOS IPSW file for virtualization
download_latest_ipsw() {
  # Installiere das ipsw-Tool, falls nicht vorhanden
  if ! command -v ipsw &> /dev/null; then
      echo "Installiere ipsw über Homebrew..."
      brew install ipsw
  fi

  # Modell für MacBook Pro (z.B. MacBookPro18,3 für 2021 14" M1 Pro)
  MODEL="MacBookPro16,3"

  # Lade die neueste IPSW für das Modell herunter
  echo "Lade die neueste IPSW für $MODEL herunter..."
  ipsw download --model "$MODEL" --latest --output "${HOME}/Downloads"

  echo "Download abgeschlossen. Die Datei befindet sich im Downloads-Ordner."
}


# Function to create the macOS VM
create_vm() {
    echo "--- Creating VM: $VM_NAME ---"
    echo "  IPSW: $IPSW_PATH"
    echo "  RAM: ${RAM_GB}GB"
    echo "  CPU: ${CPU_CORES} cores"
    echo "  Disk: ${DISK_SIZE_GB}GB"

    # utmctl create command
    # -i <ipsw_path>: Specifies the macOS IPSW restore image for installation.
    # -p macos: Specifies that it's a macOS guest.
    # -s <size_gb>: Sets the initial disk size.
    # -m <memory_mb>: Sets the RAM in MB.
    # -C <cpu_count>: Sets the number of CPU cores.
    utmctl create "$VM_NAME" \
        -i "$IPSW_PATH" \
        -p macos \
        -s "$DISK_SIZE_GB" \
        -m $((RAM_GB * 1024)) \
        -C "$CPU_CORES"

    if [ $? -ne 0 ]; then
        echo "Error: Failed to create VM '$VM_NAME'."
        exit 1
    fi
    echo "VM '$VM_NAME' created successfully."
}

# Function to start the VM
start_vm() {
    echo "--- Starting VM: $VM_NAME ---"
    # utmctl start command
    # -S: Start the VM headless (without opening the GUI window immediately).
    utmctl start "$VM_NAME" -S
    if [ $? -ne 0 ]; then
        echo "Error: Failed to start VM '$VM_NAME'."
        exit 1
    fi
    echo "VM '$VM_NAME' started successfully. The macOS installation will proceed inside the VM."
    echo "You can open the UTM application to view the VM's console."
}

# --- Main Script Logic ---

# Parse command-line arguments
START_VM_AFTER_CREATION=false
while [[ "$#" -gt 0 ]]; do
    case "$1" in
        -n|--name) VM_NAME="$2"; shift ;;
        -r|--ram) RAM_GB="$2"; shift ;;
        -c|--cpu) CPU_CORES="$2"; shift ;;
        -d|--disk) DISK_SIZE_GB="$2"; shift ;;
        -i|--ipsw) IPSW_PATH="$2"; shift ;;
        -s|--start) START_VM_AFTER_CREATION=true ;;
        -h|--help) usage ;;
        *) echo "Unknown parameter: $1"; usage ;;
    esac
    shift
done

# Check for utmctl
check_utmctl

# Validate or download IPSW path
if [ -z "$IPSW_PATH" ]; then
    echo "No IPSW path provided. Attempting to auto-download the latest macOS IPSW."
    # Call the download function and capture its output (the path)
    DOWNLOADED_IPSW_PATH=$(download_latest_ipsw)
    if [ $? -ne 0 ]; then
        echo "Automatic IPSW download failed. Please provide a valid IPSW path manually using -i."
        exit 1
    fi
    IPSW_PATH="$DOWNLOADED_IPSW_PATH"
elif [ ! -f "$IPSW_PATH" ]; then
    echo "Error: IPSW file not found at '$IPSW_PATH'."
    exit 1
fi

# Create the VM
create_vm

# Start the VM if requested
if "$START_VM_AFTER_CREATION"; then
    start_vm
else
    echo "VM '$VM_NAME' created. To start it later, run: utmctl start '$VM_NAME'"
fi

echo "Script finished."

