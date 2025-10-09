"""
Fetches the latest Windows 11 ISO download link from Microsoft's official download page using Playwright.
"""
import asyncio
from playwright.async_api import async_playwright
import random
import time
import argparse
import os

MICROSOFT_WIN11_ISO_URL = "https://www.microsoft.com/en-US/software-download/windows11"
MICROSOFT_WIN11_ARM_ISO_URL = "https://www.microsoft.com/en-us/software-download/windows11arm64"

async def fetch_win11_iso_link(arm: bool = False):
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=False)
        USER_AGENTS = [
            # Chrome on Windows
            'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
            # Chrome on Mac
            'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
            # Edge on Windows
            'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0',
            # Firefox on Windows
            'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0',
            # Safari on Mac
            'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15'
        ]
        user_agent = random.choice(USER_AGENTS)

        context = await browser.new_context(
            locale='en-US',
            timezone_id='EST',
            user_agent=user_agent
        )
        page = await context.new_page()
        url = MICROSOFT_WIN11_ARM_ISO_URL if arm else MICROSOFT_WIN11_ISO_URL
        await page.goto(url)

        async def random_mouse_movements(page, min_seconds=2, max_seconds=10):
            duration = random.uniform(min_seconds, max_seconds)
            start_time = time.time()
            box = await page.evaluate('''() => {
                const { width, height } = document.body.getBoundingClientRect();
                return { width, height };
            }''')
            width = box['width']
            height = box['height']
            while time.time() - start_time < duration:
                x = random.randint(0, int(width) - 1)
                y = random.randint(0, int(height) - 1)
                await page.mouse.move(x, y, steps=random.randint(5, 20))
                await asyncio.sleep(random.uniform(0.1, 0.5))

        await random_mouse_movements(page)

        # Accept cookies if prompted
        try:
            await page.click('button:has-text("Accept")', timeout=3000)
        except Exception:
            pass  # No cookie prompt

        await random_mouse_movements(page)

        # Select 'Windows 11 (multi-edition ISO)' from the dropdown
        # Find the option containing "Windows 11" and select it
        options = await page.query_selector_all('#product-edition option')
        for option in options:
            text = await option.text_content()
            value = await option.get_attribute('value')
            if text and "Windows 11" in text:
                await page.select_option('#product-edition', value=value)
                break
        await page.click('#submit-product-edition')
        await page.wait_for_selector('#product-languages')

        await random_mouse_movements(page)

        # Select English (or first available language)
        options = await page.query_selector_all('#product-languages option')
        lang_value = None
        for option in options:
            text = await option.text_content()
            value = await option.get_attribute('value')
            if text and "English (United States)" in text:
                lang_value = value
                break
        if lang_value is None:
            raise Exception("Could not find 'English (United States)' language option.")
        await page.select_option('#product-languages', value=lang_value)

        await random_mouse_movements(page)

        await page.click('#submit-sku')

        await random_mouse_movements(page)

        download_selector = '#download-links > div > div > a:first-child'
        await page.wait_for_selector(download_selector)
        download_button = page.locator(download_selector)

        link = await download_button.get_attribute('href')
        if not link:
            raise Exception("Could not retrieve the download link.")

        #await page.wait_for_selector('#download-links > div > div > a.first-child')
        # Get the download link
        #link = await page.get_attribute('#download-links > div > div > a.first-child', 'href')
        print(f"Windows 11 ISO download link: {link}")

        # write link to file for use in packer build
        script_dir = os.path.dirname(os.path.abspath(__file__))
        output_dir = os.path.join(script_dir, "../../vendor/windows/")
        os.makedirs(output_dir, exist_ok=True)
        if(arm):
            output_path = os.path.join(output_dir, "win11_arm_iso_url.txt")
        else:
            output_path = os.path.join(output_dir, "win11_iso_url.txt")
        with open(output_path, "w") as f:
            f.write(link)

        await browser.close()

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Fetch latest Windows 11 ISO download link.")
    parser.add_argument('--arm', action='store_true', help='Fetch Windows 11 ARM version ISO')
    args = parser.parse_args()
    asyncio.run(fetch_win11_iso_link(arm=args.arm))
