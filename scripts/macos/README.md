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
To run the script, execute the following command in your terminal:
```bash
python playwright_win11_iso.py
```
This will output the latest Windows 11 ISO download link.