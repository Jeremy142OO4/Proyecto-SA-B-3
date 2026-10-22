import json
import threading
import unittest
import urllib.request
import urllib.error
from .app import Server, State
from .test_mock import KEY, MockTests

CONTROL = 'separate-local-control-only-key'


class ControlTests(unittest.TestCase):
    def setUp(self):
        self.state = State()
        self.server = Server(self.state, KEY, control_key=CONTROL)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()

    def tearDown(self):
        self.server.shutdown(); self.server.server_close(); self.thread.join(); self.state.db.close()

    def call(self, method='GET', value=None, key=CONTROL):
        req = urllib.request.Request('http://127.0.0.1:%d/__mock/control' % self.server.server_port,
                                     method=method, headers={'X-Mock-Control-Key': key, 'Content-Type': 'application/json', 'X-Mock-Actor': 'test-admin'},
                                     data=json.dumps(value).encode() if value is not None else None)
        try:
            response = urllib.request.urlopen(req, timeout=3)
        except urllib.error.HTTPError as e:
            response = e
        with response:
            return response.code, json.load(response)

    def test_auth_and_control_not_affected_by_503(self):
        self.assertEqual(self.call(key=KEY)[0], 401)
        self.assertEqual(self.call(key='')[0], 401)
        for scenario in ('accept', 'reject', 'no_callback', 'delayed', 'unavailable', 'accept'):
            code, result = self.call('PUT', {'scenario': scenario})
            self.assertEqual((code, result['scenario'], result['target']), (200, scenario, 'MOCK'))
            self.assertNotIn(CONTROL, json.dumps(result))
        self.assertEqual(self.state.db.execute('SELECT count(*) FROM control_audit').fetchone()[0], 6)

    def test_invalid_input_cannot_change_destinations(self):
        for value in ({'scenario': 'accept', 'url': 'https://real-bank.invalid'}, {'scenario': 'unknown'}, [], {'scenario': 123}):
            self.assertEqual(self.call('PUT', value)[0], 400)
        self.assertEqual(self.call()[1]['scenario'], 'accept')

    def test_default_control_disabled(self):
        self.server.control_key = ''
        self.assertEqual(self.call()[0], 404)

    def test_separate_key_required(self):
        with self.assertRaises(ValueError):
            Server(self.state, KEY, control_key=KEY)


class SnapshotTests(MockTests):
    def test_restore_does_not_enable_suppressed_callback(self):
        self.state.control('no_callback')
        first = self.payload()
        self.post(first)
        self.state.control('accept')
        self.state.callback_url = 'http://127.0.0.1:1/interbank/v1'
        self.state.callback_key = KEY
        self.finish()
        row = self.state.row(first['interbankTransferId'])
        self.assertEqual((row['status'], row['attempts'], row['scenario']), ('ACCEPTED', 0, 'no_callback'))

    def test_change_does_not_rewrite_pending_business_result(self):
        self.state.control('reject')
        first = self.payload()
        self.post(first)
        self.state.control('accept')
        second = self.payload()
        self.post(second)
        self.finish()
        self.assertEqual(self.state.row(first['interbankTransferId'])['reason'], 'PROCESSING_ERROR')
        self.assertEqual(self.state.row(second['interbankTransferId'])['status'], 'ACCEPTED')

    def test_delay_is_captured_on_receipt(self):
        self.state.control('delayed')
        first = self.payload()
        self.post(first)
        self.state.control('accept')
        self.finish()
        self.assertEqual(self.state.row(first['interbankTransferId'])['status'], 'PROCESSING')
        self.now += 181
        self.state.process()
        self.assertEqual(self.state.row(first['interbankTransferId'])['reason'], 'EXPIRED')
