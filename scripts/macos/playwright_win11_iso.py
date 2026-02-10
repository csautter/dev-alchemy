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
MICROSOFT_WIN11_ARM_ISO_URL = (
    "https://www.microsoft.com/en-us/software-download/windows11arm64"
)

USER_AGENTS = [
    # Chrome on Windows
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",
    # Chrome on Mac
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",
    # Chrome on Linux
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",
    # Edge on Windows
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36 Edg/144.0.3719.115",
    # Edge on Mac
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36 Edg/144.0.3719.115",
    # Firefox on Windows
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0",
    # Safari on Mac
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_7_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15",
]


async def random_mouse_movements(page, min_seconds=5, max_seconds=15):
    duration = random.uniform(min_seconds, max_seconds)
    start_time = time.time()
    box = await page.evaluate(
        """() => {
            const { width, height } = document.body.getBoundingClientRect();
            return { width, height };
        }"""
    )
    width = box["width"]
    height = box["height"]
    while time.time() - start_time < duration:
        x = random.randint(0, int(width) - 1)
        y = random.randint(0, int(height) - 1)
        await page.mouse.move(x, y, steps=random.randint(5, 20))
        await asyncio.sleep(random.uniform(0.1, 0.5))


async def dismiss_modal_if_present(page):
    try:
        selector = 'button:has-text("Close")'
        await page.wait_for_selector(selector, timeout=120000)
        modal_dismiss = page.locator(selector)
        if modal_dismiss and await modal_dismiss.is_visible(timeout=30000):
            print("Dismissing modal...")
            await page.click(selector)
    except Exception:
        pass


async def click_button_with_retry(page, selector, selector_condition, retries=10):
    for _ in range(retries):
        try:
            await page.wait_for_selector(selector, timeout=60000)
            button = page.locator(selector)
            if button and await button.is_visible(timeout=30000):
                await page.click(selector)

            else:
                print(f"{selector} button is not visible.")
        except Exception:
            pass

        try:
            await page.wait_for_selector(selector_condition)
        except Exception:
            pass

        expected = page.locator(selector_condition)
        if expected and await expected.is_visible(timeout=30000):
            return True
        else:
            await dismiss_modal_if_present(page)

    return False


async def select_option_by_text(page, selector, text_match):
    options = await page.query_selector_all(f"{selector} option")
    for option in options:
        text = await option.text_content()
        value = await option.get_attribute("value")
        if text and text_match in text:
            await page.select_option(selector, value=value)
            return value
    return None


async def fetch_win11_iso_link(arm: bool = False, headless: bool = False):
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=headless)
        user_agent = random.choice(USER_AGENTS)
        context = await browser.new_context(
            locale="en-US", timezone_id="EST", user_agent=user_agent
        )
        page = await context.new_page()
        url = MICROSOFT_WIN11_ARM_ISO_URL if arm else MICROSOFT_WIN11_ISO_URL
        await page.goto(url)

        await random_mouse_movements(page)

        # Accept cookies if prompted
        try:
            await page.click('button:has-text("Accept")', timeout=3000)
        except Exception:
            pass

        await random_mouse_movements(page)

        # Select 'Windows 11 (multi-edition ISO)' from the dropdown
        await select_option_by_text(page, "#product-edition", "Windows 11")

        expect_creation_of_selector = "#product-languages"
        await click_button_with_retry(
            page, "#submit-product-edition", expect_creation_of_selector
        )

        await random_mouse_movements(page)

        # Select English (United States)
        lang_value = await select_option_by_text(
            page, "#product-languages", "English (United States)"
        )
        if lang_value is None:
            raise Exception("Could not find 'English (United States)' language option.")

        await random_mouse_movements(page)

        # expected selector
        download_selector = "#download-links > div > div > a:first-child"
        await click_button_with_retry(page, "#submit-sku", download_selector)

        await random_mouse_movements(page)

        await page.wait_for_selector(download_selector, timeout=60000)
        download_button = page.locator(download_selector)
        link = await download_button.get_attribute("href")
        if not link:
            raise Exception("Could not retrieve the download link.")

        print(f"Windows 11 ISO download link: {link}")

        # Write link to file for use in packer build
        script_dir = os.path.dirname(os.path.abspath(__file__))
        output_dir = os.path.join(script_dir, "../../vendor/windows/")
        os.makedirs(output_dir, exist_ok=True)
        output_path = os.path.join(
            output_dir, "win11_arm64_iso_url.txt" if arm else "win11_amd64_iso_url.txt"
        )
        with open(output_path, "w") as f:
            f.write(link)

        await browser.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Fetch latest Windows 11 ISO download link."
    )
    parser.add_argument(
        "--arm", action="store_true", help="Fetch Windows 11 ARM version ISO"
    )
    parser.add_argument(
        "--headless",
        type=lambda x: (str(x).lower() == "true"),
        default=True,
        help="Run browser in headless mode (true/false, default: true)",
    )
    args = parser.parse_args()
    asyncio.run(fetch_win11_iso_link(arm=args.arm, headless=args.headless))
