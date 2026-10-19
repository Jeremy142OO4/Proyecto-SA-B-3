import importlib.util
from pathlib import Path
import unittest
import sys

sys.dont_write_bytecode = True

spec = importlib.util.spec_from_file_location('interbank_config', Path(__file__).with_name('configure-interbank.py'))
config = importlib.util.module_from_spec(spec)
spec.loader.exec_module(config)


class InterbankConfigurationTest(unittest.TestCase):
    def test_receive_only_does_not_require_partner_url_or_key(self):
        secret, settings = config.resources({'KUBE_NAMESPACE': 'test', 'INTERBANK_ENABLED': 'true', 'INTERBANK_API_KEY': 'local-only-inbound-key'})
        self.assertIn('INTERBANK_ENABLED=true', settings)
        self.assertIn('INTERBANK_PARTNER_BASE_URL=', settings)
        self.assertNotIn('local-only-inbound-key', ' '.join(settings))
        self.assertEqual(secret['stringData']['INTERBANK_API_KEY'], 'local-only-inbound-key')

    def test_default_disabled(self):
        _, settings = config.resources({'KUBE_NAMESPACE': 'test'})
        self.assertIn('INTERBANK_ENABLED=false', settings)

    def test_enabled_requires_inbound_authentication(self):
        with self.assertRaises(ValueError):
            config.resources({'KUBE_NAMESPACE': 'test', 'INTERBANK_ENABLED': 'true'})

    def test_outbound_url_still_requires_https_and_fixed_path(self):
        for url in ['http://bank14.test/interbank/v1', 'https://bank14.test/other', 'https://u:p@bank14.test/interbank/v1']:
            with self.subTest(url=url), self.assertRaises(ValueError):
                config.resources({'KUBE_NAMESPACE': 'test', 'INTERBANK_PARTNER_BASE_URL': url})


if __name__ == '__main__':
    unittest.main()
