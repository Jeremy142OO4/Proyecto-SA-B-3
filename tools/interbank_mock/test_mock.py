from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import tempfile
import threading
import unittest
import urllib.error
import urllib.request
import uuid

from .app import Server, State, utc
from .config import BankSettings, base_url
from .contract import CONTRACT, CONTRACT_PATH, CONTRACT_SHA256, check, response_schema, validate

KEY = 'local-mock-test-key-not-a-real-bank-key'
TRACE = '00-'+'a'*32+'-'+'b'*16+'-01'


class MockTests(unittest.TestCase):
    def setUp(self):
        self.now = 1800000000.0
        self.state = State(clock=lambda: self.now)
        self.server = Server(self.state, KEY)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.url = 'http://127.0.0.1:'+str(self.server.server_port)+'/interbank/v1'

    def tearDown(self):
        self.server.shutdown(); self.server.server_close(); self.thread.join()
        self.state.db.close()

    def payload(self, **changes):
        result = dict(interbankTransferId=str(uuid.uuid4()), originBankCode='BANK03', destinationBankCode='BANK14',
                      originAccount='TEST-BANK03-001', destinationAccount='TEST-BANK14-001', amount='150.00',
                      currency='GTQ', createdAt=utc(self.now), expiresAt=utc(self.now+180))
        return dict(result, **changes)

    def api(self, method, path, body=None, headers=None, root=None):
        h = {'X-Api-Key': KEY, 'traceparent': TRACE, 'Content-Type': 'application/json'}
        if body and 'interbankTransferId' in body:
            h['Idempotency-Key'] = body['interbankTransferId']
        h.update(headers or {})
        req = urllib.request.Request((root or self.url)+path, headers=h, method=method,
                                     data=json.dumps(body).encode() if body is not None else None)
        try:
            response = urllib.request.urlopen(req, timeout=3)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            return response.status, json.load(response)

    def post(self, payload=None):
        return self.api('POST', '/transfers', payload or self.payload())

    def finish(self):
        self.now += 1
        self.state.process()

    def test_original_and_all_endpoints_exist(self):
        self.assertEqual(hashlib.sha256(CONTRACT_PATH.read_bytes()).hexdigest(), CONTRACT_SHA256)
        self.assertEqual(len(CONTRACT['paths']), 8)

    def test_public_health_and_auth(self):
        code, value = self.api('GET', '/health', headers={'X-Api-Key': '', 'traceparent': ''})
        self.assertEqual((code, value['bankCode']), (200, 'BANK14'))
        self.assertEqual(self.api('GET', '/reconciliation?date=2027-01-15', headers={'X-Api-Key': 'wrong'})[0], 401)
        self.assertEqual(self.api('GET', '/reconciliation?date=2027-01-15', headers={'traceparent': ''})[0], 400)
        self.assertEqual(self.api('GET', '/reconciliation?date=2027-01-15', headers={'traceparent': '00-'+'0'*32+'-'+'b'*16+'-01'})[0], 400)

    def test_receive_process_and_response_schemas(self):
        payload = self.payload()
        code, ack = self.post(payload)
        self.assertEqual(code, 202)
        validate(response_schema('/transfers', 'post', code), ack)
        self.finish()
        code, status = self.api('GET', '/transfers/'+payload['interbankTransferId'])
        self.assertEqual(status['status'], 'ACCEPTED')
        check('TransferStatus', status)
        self.assertEqual(self.state.db.execute('SELECT COUNT(*) FROM credits').fetchone()[0], 1)

    def test_duplicate_conflict_and_concurrent_idempotence(self):
        from concurrent.futures import ThreadPoolExecutor
        payload = self.payload()
        with ThreadPoolExecutor(max_workers=6) as pool:
            results = list(pool.map(lambda _: self.post(payload)[0], range(6)))
        self.assertEqual(results.count(202), 1)
        self.assertEqual(results.count(200), 5)
        self.finish()
        self.assertEqual(self.post(payload)[0], 200)
        self.assertEqual(self.post(dict(payload, amount='151.00'))[0], 409)
        self.assertEqual(self.state.db.execute('SELECT COUNT(*) FROM credits').fetchone()[0], 1)

    def test_expiration_before_receipt_and_before_processing(self):
        payload = self.payload(createdAt=utc(self.now-181), expiresAt=utc(self.now-1))
        self.assertEqual(self.post(payload)[0], 422)
        self.assertIsNone(self.state.row(payload['interbankTransferId']))
        payload = self.payload()
        self.post(payload); self.now += 181; self.state.process()
        self.assertEqual(self.state.row(payload['interbankTransferId'])['reason'], 'EXPIRED')
        self.assertEqual(self.state.db.execute('SELECT COUNT(*) FROM credits').fetchone()[0], 0)

    def test_contract_validation(self):
        for change in [dict(destinationAccount='a'*35), dict(originBankCode='BANK3'), dict(amount=150.0),
                       dict(currency='USD'), dict(expiresAt=utc(self.now+181)), dict(extra='invalid')]:
            with self.subTest(change=change):
                self.assertEqual(self.post(self.payload(**change))[0], 400)
        payload = self.payload()
        self.assertEqual(self.api('POST', '/transfers', payload, {'Idempotency-Key': str(uuid.uuid4())})[0], 400)

    def test_business_rejections_and_unavailable(self):
        for change, reason in [(dict(destinationAccount='UNKNOWN'), 'ACCOUNT_NOT_FOUND'),
                               (dict(amount='10000.01'), 'AMOUNT_LIMIT_EXCEEDED'),
                               (dict(amount='0.00'), 'AMOUNT_LIMIT_EXCEEDED')]:
            payload = self.payload(**change)
            self.assertEqual(self.post(payload)[0], 202)
            self.finish()
            self.assertEqual(self.state.row(payload['interbankTransferId'])['reason'], reason)
        self.state.scenario = 'reject'
        payload = self.payload(); self.post(payload); self.finish()
        self.assertEqual(self.state.row(payload['interbankTransferId'])['reason'], 'PROCESSING_ERROR')
        self.state.scenario = 'unavailable'
        self.assertEqual(self.post()[0], 503)

    def test_cancel_tombstone_and_terminal_results(self):
        identifier = str(uuid.uuid4())
        cancel = dict(originBankCode='BANK03', reason='TIMEOUT')
        route = '/transfers/'+identifier+'/cancel'
        self.assertEqual(self.api('POST', route, cancel)[1]['result'], 'CANCELLED')
        payload = self.payload(interbankTransferId=identifier)
        self.assertEqual(self.post(payload)[1]['status'], 'CANCELLED')
        self.finish()
        self.assertEqual(self.state.db.execute('SELECT COUNT(*) FROM credits').fetchone()[0], 0)
        payload = self.payload(); self.post(payload); self.finish()
        self.assertEqual(self.api('POST', '/transfers/'+payload['interbankTransferId']+'/cancel', cancel)[1]['result'], 'ALREADY_ACCEPTED')
        payload = self.payload(destinationAccount='missing'); self.post(payload); self.finish()
        self.assertEqual(self.api('POST', '/transfers/'+payload['interbankTransferId']+'/cancel', cancel)[1]['result'], 'ALREADY_REJECTED')
        self.assertEqual(self.api('POST', route, dict(cancel, originBankCode='BANK99'))[0], 401)

    def test_reconciliation_pagination_and_reversal_contract(self):
        payloads = [self.payload() for _ in range(3)]
        for payload in payloads:
            self.post(payload)
        self.finish()
        day = utc(self.now)[:10]
        _, page = self.api('GET', '/reconciliation?date='+day+'&pageSize=2&page=2')
        check('ReconciliationPage', page)
        self.assertEqual((page['totalItems'], len(page['items'])), (3, 1))
        self.assertEqual(self.api('GET', '/reconciliation?date='+day+'&pageSize=501')[0], 400)
        reversal = dict(reversalId=str(uuid.uuid4()), originalTransferId=payloads[0]['interbankTransferId'],
                        originBankCode='BANK03', amount='150.00', currency='GTQ')
        code, value = self.api('POST', '/reversals', reversal)
        self.assertEqual(code, 202); check('ReversalStatus', value)
        self.assertEqual(self.api('GET', '/reversals/'+reversal['reversalId'])[1]['status'], 'PROCESSING')

    def test_callback_acknowledgement_exact_contract(self):
        payload = self.payload(originBankCode='BANK14', destinationBankCode='BANK03')
        self.state.register_outbound(payload)
        result = dict(interbankTransferId=payload['interbankTransferId'], destinationBankCode='BANK03',
                      status='ACCEPTED', processedAt=utc(self.now))
        route = '/transfers/'+payload['interbankTransferId']+'/result'
        code, ack = self.api('POST', route, result)
        self.assertEqual(code, 200)
        self.assertEqual(ack, {'interbankTransferId': payload['interbankTransferId'], 'received': True})
        self.assertEqual(self.api('POST', route, result)[1], ack)
        self.assertEqual(self.api('POST', '/transfers/'+str(uuid.uuid4())+'/result', result)[0], 404)

    def test_persistent_state_after_restart(self):
        with tempfile.TemporaryDirectory() as directory:
            db = str(Path(directory)/'state.sqlite3')
            first = State(db, clock=lambda: self.now)
            payload = self.payload()
            first.dispatch('POST', '/transfers', payload, {'Idempotency-Key': payload['interbankTransferId'], 'traceparent': TRACE}, {})
            first.db.close()
            second = State(db, clock=lambda: self.now+1)
            second.process()
            self.assertEqual(second.row(payload['interbankTransferId'])['status'], 'ACCEPTED')
            self.assertEqual(second.dispatch('POST', '/transfers', payload, {'Idempotency-Key': payload['interbankTransferId'], 'traceparent': TRACE}, {})[0], 200)
            second.db.close()

    def test_real_callback_http_and_retry(self):
        receiver = State(bank='BANK03', partner='BANK14', clock=lambda: self.now)
        server = Server(receiver, KEY)
        thread = threading.Thread(target=server.serve_forever, daemon=True); thread.start()
        try:
            payload = self.payload()
            receiver.register_outbound(payload)
            self.state.callback_url = 'http://127.0.0.1:'+str(server.server_port)+'/interbank/v1'
            self.state.callback_key = 'wrong-key'
            self.post(payload); self.finish()
            row = self.state.row(payload['interbankTransferId'])
            self.assertEqual(row['attempts'], 1)
            self.assertEqual(row['callback_due'], self.now+2)
            self.state.callback_key = KEY; self.now += 2; self.state.process()
            row = self.state.row(payload['interbankTransferId'])
            self.assertIsNone(row['callback_due'])
            result = receiver.db.execute('SELECT result FROM outgoing WHERE id=?', (payload['interbankTransferId'],)).fetchone()[0]
            self.assertEqual(json.loads(result)['status'], 'ACCEPTED')
        finally:
            server.shutdown(); server.server_close(); thread.join(); receiver.db.close()

    def test_callbacks_stop_after_24_hours_and_no_callback_scenario(self):
        payload = self.payload()
        self.state.scenario = 'no_callback'
        self.post(payload); self.finish()
        self.assertEqual(self.state.row(payload['interbankTransferId'])['status'], 'ACCEPTED')
        self.assertEqual(self.state.row(payload['interbankTransferId'])['attempts'], 0)
        self.state.scenario = 'accept'
        self.state.callback_url = 'http://127.0.0.1:1/interbank/v1'
        self.state.callback_key = KEY
        self.now += 86401
        self.state.process()
        self.assertIsNone(self.state.row(payload['interbankTransferId'])['callback_due'])
        self.assertEqual(self.state.row(payload['interbankTransferId'])['attempts'], 0)


class ConfigurationTests(unittest.TestCase):
    def test_mock_cannot_call_remote_bank(self):
        with self.assertRaises(ValueError):
            State(callback_url='https://bank03.example/interbank/v1', callback_key=KEY)

    def test_real_settings_https_and_secret_redaction(self):
        env = dict(INTERBANK_PUBLIC_BASE_URL='https://bank03.example/interbank/v1',
                   INTERBANK_PARTNER_BASE_URL='https://bank14.example/interbank/v1',
                   INTERBANK_API_KEY=KEY, INTERBANK_PARTNER_API_KEY=KEY+'partner')
        settings = BankSettings.from_env(env)
        self.assertEqual((settings.bank_code, settings.partner_code), ('BANK03', 'BANK14'))
        self.assertNotIn(KEY, repr(settings))
        for url in ['http://example.com/interbank/v1', 'https://u:p@example.com/interbank/v1',
                    'https://example.com/interbank/v1?key=secret', 'https://<HOST>/interbank/v1']:
            with self.assertRaises(ValueError):
                base_url(url)
        self.assertEqual(base_url('http://127.0.0.1/interbank/v1', True), 'http://127.0.0.1/interbank/v1')
        with self.assertRaises(ValueError):
            BankSettings.from_env({})


if __name__ == '__main__':
    unittest.main()
