import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch

HERE = Path(__file__).parent
def module(name):
    spec = importlib.util.spec_from_file_location(name, HERE / (name + '.py'))
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


class SafetyTests(unittest.TestCase):
    def test_fixture_requires_bare_https_origin(self):
        api = module('prepare_fixture').API
        for value in ('http://bank.example', 'https://bank.example/api'):
            with self.assertRaises(ValueError):
                api(value)

    def test_fixture_does_not_follow_auth_redirect(self):
        api = module('prepare_fixture').API('https://bank.example')
        redirect = next(h for h in api.opener.handlers if h.__class__.__name__ == 'NoRedirect')
        self.assertIsNone(redirect.redirect_request(None, None, 302, '', {}, 'https://other.example'))

    def test_observation_preserves_hpa_evidence_without_sensitive_pod_config(self):
        observe = module('observe_hpa')
        def fetch(path):
            if path.endswith('horizontalpodautoscalers'):
                return {'items':[{'metadata':{'name':'transaction-service'},
                    'spec':{'scaleTargetRef':{'name':'transaction-service'},'minReplicas':1,'maxReplicas':4,
                            'metrics':[{'type':'Resource','resource':{'name':'cpu','target':{'averageUtilization':80}}}]},
                    'status':{'currentReplicas':1,'desiredReplicas':2,'currentMetrics':[]}}]}
            if path.endswith('deployments'):
                return {'items':[]}
            return {'items':[{'metadata':{'name':'pod-2','uid':'uid','creationTimestamp':'2026-10-23T00:00:00Z',
                                         'labels':{'app':'transaction-service'}},
                             'spec':{'env':[{'PASSWORD':'never export'}]},
                             'status':{'phase':'Running','containerStatuses':[{'name':'app','ready':True,'restartCount':0}]}}]}
        result = observe.snapshot(fetch, 'bank-usac')
        self.assertEqual(result['hpas'][0]['status']['desiredReplicas'], 2)
        self.assertEqual(result['pods'][0]['uid'], 'uid')
        self.assertNotIn('spec', result['pods'][0])
        self.assertNotIn('PASSWORD', str(result))


if __name__ == '__main__':
    unittest.main()
