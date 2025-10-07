# MacOS Setup Instructions
## playwright_win11_iso.py
This script uses Playwright to fetch the latest Windows 11 ISO download link from Microsoft's official download page.
### Prerequisites
- Python 3.7 or higher
- Playwright library
### Installation
1. **Install Python**: Ensure you have Python 3.7 or higher installed. You can install it via Homebrew:
   ```bash
   brew install python
   ```
2. **Set Up a Virtual Environment** (optional but recommended):
   ```bash
   python3 -m venv .venv
   source .venv/bin/activate
   ```
3. **Install Playwright**:
   ```bash
   pip install playwright
   python -m playwright install
   ```
### Running the Script
Activate your virtual environment if you created one:
```bash
source .venv/bin/activate
```
To get the latest Windows 11 ISO download link, run:
```bash
python playwright_win11_iso.py
```
To get the latest Windows 11 ISO arm version download link, use:
```bash
python playwright_win11_iso.py --arm
```
This will output the latest Windows 11 ISO download link in the terminal.
Additionally , the script saves the download link to a file named `./vendor/windows/win11_iso_url.txt` or `./vendor/windows/win11_arm_iso_url.txt`.

### Download the ISO
You can use `curl` or `wget` to download the ISO using the link saved in the file:
```bash
cd ./vendor/windows/
curl --progress-bar -o win11_25h2_english_x64.iso $(cat ./win11_iso_url.txt)
```
or for arm:
```bash
cd ./vendor/windows/
curl --progress-bar -o win11_25h2_english_arm64.iso $(cat ./win11_arm_iso_url.txt)
```