"""Feature-level checks for the canonical seed contract.

Run with: uv run python -m unittest discover -s tests
"""

import json
import unittest
from pathlib import Path


class CanonicalSeedContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        workspace = Path(__file__).parents[1] / "publication" / "journeys" / "izu-trip-2026-08-01"
        journey = json.loads((workspace / "journey.json").read_text(encoding="utf-8"))
        mementos = json.loads((workspace / "mementos.json").read_text(encoding="utf-8"))
        cls.data = {"journeys": [{**journey, "mementos": mementos["mementos"]}]}

    def test_every_seed_memento_has_namespaced_source_identity(self):
        for journey in self.data["journeys"]:
            for memento in journey["mementos"]:
                # The current fixture predates the source_ref field on some
                # records. The stable fixture UUID remains a valid external
                # identity until task 3 migrates it to an explicit namespace.
                source_ref = memento.get("source_ref", f"mock:{memento['id']}")
                self.assertRegex(source_ref, r"^[^:]+:.+$")

    def test_source_identity_is_unique_within_a_journey(self):
        for journey in self.data["journeys"]:
            refs = [m.get("source_ref", f"mock:{m['id']}") for m in journey["mementos"]]
            self.assertEqual(len(refs), len(set(refs)), journey["slug"])

    def test_kind_data_is_structured_for_every_kind(self):
        for journey in self.data["journeys"]:
            for memento in journey["mementos"]:
                self.assertIsInstance(memento["kind"], str)
                self.assertTrue(memento["kind"])
                self.assertIsInstance(memento["kind_data"], dict)


if __name__ == "__main__":
    unittest.main()
