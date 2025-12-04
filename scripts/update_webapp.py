#!/usr/bin/env python3
"""Re-sync the upstream fast.com webapp assets and patch them for
self-hosted deployments, while reusing assets from the original webapp.

- Downloads index.html, app js, app css, localization json and oc-webfont
  - There is no CORS support for the font
- Patches the app js bundle to use local endpoints instead of Netflix endpoints.
- Patches the app css bundle to use remote assets (https://fast.com/assets/...).
- Patches index.html to use local js/css/localized json; other URLs point to fast.com.
"""

import re
import shutil
import sys
from html.parser import HTMLParser
from pathlib import Path
from typing import List, Optional, Sequence, Tuple
from urllib.parse import urljoin
from urllib.request import Request, urlopen

FAST_ROOT_URL = "https://fast.com/"
CWD = Path.cwd()
WEBAPP_DIR = CWD / "original-webapp"

# Regex to match app bundles (e.g., /app-07ba96.css, /app-0bffe1.js)
APP_ASSET_REGEX = re.compile(r"^/?app-[0-9a-fA-F]+\.(js|css)$")

# A minimal list of extra assets that need to be downloaded due to CORS restrictions
FIXED_ASSETS = [
    "/assets/fonts/oc-webfont.min.css",
    "/assets/fonts/fonts/oc-webfont.svg",
    "/assets/fonts/fonts/oc-webfont.ttf",
    "/assets/fonts/fonts/oc-webfont.woff",
    "/assets/fonts/fonts/oc-webfont.eot",
]


class AppAssetRefs(HTMLParser):
    """HTML parser to collect local assets from index.html."""

    def __init__(self) -> None:
        super().__init__()
        self.js_path: Optional[str] = None
        self.css_path: Optional[str] = None
        self.localized_path: Optional[str] = None

        # Configuration: tag -> [(attribute, validation_regex, target_field, description), (...)]
        self._rules = {
            "script": [("src", APP_ASSET_REGEX, "js_path", "JS bundle")],
            "link": [
                ("href", APP_ASSET_REGEX, "css_path", "CSS bundle"),
            ],
            "body": [("localized", None, "localized_path", "localized JSON")],
        }

    def handle_starttag(self, tag: str, attrs: List[Tuple[str, Optional[str]]]) -> None:
        if tag not in self._rules:
            return

        attrs_dict = dict(attrs)
        for attr_name, pattern, target_field, log_desc in self._rules[tag]:
            value = attrs_dict.get(attr_name)

            # Check if attribute exists and matches pattern (if pattern is provided)
            if value and (pattern is None or pattern.match(value)):
                print(f"[parse] found {log_desc}: {value}")
                setattr(self, target_field, value)

    def validate(self) -> None:
        """Raise an error if any required asset is missing."""
        missing = []
        if not self.js_path:
            missing.append("app JS bundle (app-*.js)")
        if not self.css_path:
            missing.append("app CSS bundle (app-*.css)")
        if not self.localized_path:
            missing.append("localized JSON (localized*.json)")
        if missing:
            raise RuntimeError(
                f"Failed to find required assets in index.html: {', '.join(missing)}"
            )


def http_get(url: str, destination: Path) -> str:
    """Fetch URL content and write to destination. Returns content as string."""
    print(f"[fetch] {url} -> {destination}")
    req = Request(url, headers={"User-Agent": "fast-clone-fetcher/0.1"})
    with urlopen(req, timeout=30) as resp:  # nosec: fixed target fast.com
        if resp.status != 200:
            raise RuntimeError(f"GET {url} -> HTTP {resp.status}")
        raw_data = resp.read()
        destination.write_bytes(raw_data)
        return raw_data.decode("utf-8", errors="replace")


def fetch_asset(asset_path: str, dest_dir: Path) -> Path:
    """Fetch an asset from fast.com and save it locally. Returns local path."""
    url = urljoin(FAST_ROOT_URL, asset_path)
    local_path = dest_dir / Path(asset_path).name

    http_get(url, local_path)

    return local_path


def patch_file(path: Path, replacements: Sequence[Tuple[str, str]]) -> None:
    """Apply string replacements to a file."""
    if not path.exists():
        print(f"[patch] {path.name}: file not found; skipping", file=sys.stderr)
        return

    content = path.read_text(encoding="utf-8", errors="replace")
    original_content = content

    for old, new in replacements:
        if old in content:
            count = content.count(old)
            content = content.replace(old, new)
            print(f"[patch] {path.name}: replaced {count}x: {old!r}")

    if content != original_content:
        path.write_text(content, encoding="utf-8")


def fetch_assets() -> Tuple[Path, Path]:
    """Fetch all required assets from fast.com. Returns (js_path, css_path)."""
    # Clean and recreate output directory
    shutil.rmtree(WEBAPP_DIR, ignore_errors=True)
    WEBAPP_DIR.mkdir(parents=True, exist_ok=True)

    # Fetch and parse index.html
    index_path = WEBAPP_DIR / "index.html"
    html_content = http_get(FAST_ROOT_URL, index_path)

    parser = AppAssetRefs()
    parser.feed(html_content)
    parser.validate()

    js_path = fetch_asset(parser.js_path, WEBAPP_DIR)
    css_path = fetch_asset(parser.css_path, WEBAPP_DIR)
    fetch_asset(parser.localized_path, WEBAPP_DIR)  # fetched but not passed around

    # Some fixed assets.
    for asset in FIXED_ASSETS:
        dest = WEBAPP_DIR / Path(asset.lstrip("/")).parent
        print(f"[fetch] {asset} -> {dest}")
        dest.mkdir(parents=True, exist_ok=True)
        fetch_asset(asset, dest)

    return js_path, css_path


def main() -> None:
    print(f"[update-webapp] Fetching upstream assets from fast.com to {WEBAPP_DIR}")
    js_file, css_file = fetch_assets()

    print("[update-webapp] Patching JS bundle (API endpoints)...")
    patch_file(
        js_file,
        [
            ('"api.fast.com/netflix/speedtest/v2"', '"/netflix/speedtest/v2"'),
            ('"api-global.netflix.com/oca/speedtest"', '"/api/oca/speedtest"'),
            ('"https://ichnaea-web.netflix.com/cl2"', '"/telemetry/cl2"'),
            ('"https://ichnaea.test.netflix.com/cl2"', '"/telemetry-test/cl2"'),
            ('url="https://"+endpoint+"', 'url=endpoint+"'),
        ],
    )

    print("[update-webapp] Patching CSS bundle (asset URLs)...")
    patch_file(
        css_file,
        [("url(/assets/", "url(https://fast.com/assets/")],
    )

    print("[update-webapp] Patching index.html...")
    patch_file(
        WEBAPP_DIR / "index.html",
        [
            ('="/assets/', '="https://fast.com/assets/'),
            ('="https://fast.com/assets/fonts/', '="/assets/fonts/'),
            ('href="https://fast.com" rel="canonical"', 'href="/" rel="canonical"'),
        ],
    )

    print("[update-webapp] Done!")


if __name__ == "__main__":  # pragma: no cover
    main()
