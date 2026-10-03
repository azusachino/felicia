import json
import os
import selectors
import subprocess
import unittest
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


class BrowserMockTest(unittest.TestCase):
    def test_node_mock_serves_synthetic_routes_on_loopback(self):
        process = subprocess.Popen(
            ["bun", "scripts/browser_preview_mock.ts"], cwd=ROOT,
            env={**os.environ, "BROWSER_MOCK_PORT": "0"},
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
        )
        try:
            with selectors.DefaultSelector() as selector:
                selector.register(process.stdout, selectors.EVENT_READ)
                self.assertTrue(selector.select(15), "Node mock did not become ready")
                line = process.stdout.readline().strip()
            self.assertTrue(line.startswith("Felicia browser mock listening on http://127.0.0.1:"), line)
            base = line.split(" on ", 1)[1]
            for path in ("site.json", "journeys.json", "journeys/browser-journey.json", "journeys/browser-journey/mementos.json"):
                with self.subTest(path=path), urllib.request.urlopen(f"{base}/api/v1/{path}", timeout=5) as response:
                    self.assertEqual(response.status, 200)
                    self.assertEqual(response.headers["Access-Control-Allow-Origin"], "*")
                    self.assertTrue(json.load(response))
            with self.assertRaises(urllib.error.HTTPError) as error:
                urllib.request.urlopen(f"{base}/not-a-route", timeout=5)
            self.assertEqual(error.exception.code, 404)
            error.exception.close()
        finally:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
            process.stdout.close()
            process.stderr.close()
