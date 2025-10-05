"""
Fetches the latest Windows 11 ISO download link from Microsoft's official download page using Playwright.
"""
import asyncio
from playwright.async_api import async_playwright

MICROSOFT_WIN11_ISO_URL = "https://www.microsoft.com/en-US/software-download/windows11"

async def fetch_win11_iso_link():
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=False)
        context = await browser.new_context(
            locale='en-US',
            timezone_id='EST',
        )
        page = await context.new_page()
        await page.goto(MICROSOFT_WIN11_ISO_URL)

        # Accept cookies if prompted
        try:
            await page.click('button:has-text("Accept")', timeout=3000)
        except Exception:
            pass  # No cookie prompt

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

        # Select English (or first available language)
        options = await page.query_selector_all('#product-languages option')
        lang_value = None
        for option in options:
            text = await option.text_content()
            value = await option.get_attribute('value')
            if text and "English International" in text:
                lang_value = value
                break
        if lang_value is None:
            raise Exception("Could not find 'English International' language option.")
        await page.select_option('#product-languages', value=lang_value)

        await page.click('#submit-sku')

        download_selector = '#download-links > div > div > a.first-child'
        download_button = page.locator(download_selector)
        await download_button.wait_for(state='visible', timeout=10000)
        if not download_button:
            raise Exception("Download button not found.")

        link = await download_button.get_attribute('href')
        if not link:
            raise Exception("Could not retrieve the download link.")

        #await page.wait_for_selector('#download-links > div > div > a.first-child')
        # Get the download link
        #link = await page.get_attribute('#download-links > div > div > a.first-child', 'href')
        print(f"Windows 11 ISO download link: {link}")
        await browser.close()

if __name__ == "__main__":
    asyncio.run(fetch_win11_iso_link())
