import json
import unittest
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker


ROOT = Path(__file__).resolve().parents[1]
SCHEMA = ROOT / "contracts" / "canonical" / "v1" / "schema.json"
EXAMPLE = ROOT / "contracts" / "canonical" / "v1" / "examples" / "memento.json"


class CanonicalContractTest(unittest.TestCase):
    def test_canonical_example_is_valid(self):
        schema = json.loads(SCHEMA.read_text(encoding="utf-8"))
        document = json.loads(EXAMPLE.read_text(encoding="utf-8"))
        errors = list(Draft202012Validator(schema, format_checker=FormatChecker()).iter_errors(document))
        self.assertEqual([], errors)

    def test_canonical_media_requires_visibility_and_source(self):
        schema = json.loads(SCHEMA.read_text(encoding="utf-8"))
        document = json.loads(EXAMPLE.read_text(encoding="utf-8"))
        del document["data"]["media"][0]["visibility"]
        media_schema = {"$schema": schema["$schema"], "$defs": schema["$defs"], "$ref": "#/$defs/media"}
        errors = list(Draft202012Validator(media_schema, format_checker=FormatChecker()).iter_errors(document["data"]["media"][0]))
        self.assertTrue(any(error.validator == "required" for error in errors))


if __name__ == "__main__":
    unittest.main()
