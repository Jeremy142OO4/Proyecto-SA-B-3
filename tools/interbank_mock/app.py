"""Loopback-only contract mock; SQLite records are fictitious, never real funds."""
import argparse
from datetime import datetime, timezone
from decimal import Decimal
import hmac
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import re
import secrets
import sqlite3
import ssl
import threading
import time
from urllib.parse import parse_qs, urlsplit
import urllib.request
import uuid

from .contract import CONTRACT, check, timestamp, validate, response_schema
from .config import base_url


def utc(seconds=None):
    return datetime.fromtimestamp(time.time() if seconds is None else seconds, timezone.utc).isoformat().replace('+00:00', 'Z')


class Fault(Exception):
    def __init__(self, status, code, message):
        self.status, self.code, self.message = status, code, message


class State:
    def __init__(self, database=':memory:', bank='BANK14', partner='BANK03', accounts=('TEST-BANK14-001',),
                 delay=0.1, scenario='accept', callback_url='', callback_key='', clock=time.time):
        if not re.fullmatch(r'BANK[0-9]{2}', bank) or not re.fullmatch(r'BANK[0-9]{2}', partner) or bank == partner:
            raise ValueError('Distinct BANKxx codes required')
        if delay < 0 or scenario not in ('accept', 'reject', 'no_callback', 'unavailable'):
            raise ValueError('Invalid mock scenario or processing delay')
        for account in accounts:
            check('AccountNumber', account)
        if callback_url:
            callback_url = base_url(callback_url, allow_loopback_http=True)
            if urlsplit(callback_url).hostname not in ('127.0.0.1', 'localhost', '::1'):
                raise ValueError('The local mock may only call loopback callback targets')
            if not callback_key:
                raise ValueError('Callback key required')
        self.bank, self.partner, self.accounts = bank, partner, set(accounts)
        self.delay, self.scenario, self.clock = delay, scenario, clock
        self.callback_url, self.callback_key = callback_url, callback_key
        self.lock = threading.RLock()
        self.db = sqlite3.connect(database, check_same_thread=False)
        self.db.row_factory = sqlite3.Row
        self.db.executescript('''
        CREATE TABLE IF NOT EXISTS transfers(id TEXT PRIMARY KEY, body TEXT, status TEXT, reason TEXT,
          received TEXT, updated TEXT, due REAL, trace TEXT, callback_due REAL, attempts INTEGER DEFAULT 0);
        CREATE TABLE IF NOT EXISTS credits(id TEXT PRIMARY KEY, account TEXT, amount TEXT);
        CREATE TABLE IF NOT EXISTS reversals(id TEXT PRIMARY KEY, body TEXT, status TEXT);
        CREATE TABLE IF NOT EXISTS outgoing(id TEXT PRIMARY KEY, body TEXT, result TEXT);
        ''')

    def row(self, identifier):
        return self.db.execute('SELECT * FROM transfers WHERE id=?', (identifier,)).fetchone()

    def status(self, row):
        result = dict(interbankTransferId=row['id'], status=row['status'], updatedAt=row['updated'])
        if row['reason']:
            result['reasonCode'] = row['reason']
        return result

    def register_outbound(self, payload):
        """Test fixture for callback validation, not an exposed creation endpoint."""
        check('TransferRequest', payload)
        if payload['originBankCode'] != self.bank or payload['destinationBankCode'] != self.partner:
            raise ValueError('Wrong outbound fixture direction')
        with self.lock, self.db:
            self.db.execute('INSERT INTO outgoing VALUES(?,?,NULL)', (payload['interbankTransferId'], json.dumps(payload, sort_keys=True)))

    def dispatch(self, method, path, body, headers, query):
        with self.lock, self.db:
            return self._dispatch(method, path, body, headers, query)

    def _dispatch(self, method, path, body, headers, query):
        if path == '/health' and method == 'GET':
            if self.scenario == 'unavailable':
                raise Fault(503, 'SERVICE_UNAVAILABLE', 'Mock unavailable scenario')
            return 200, {'status': 'UP', 'bankCode': self.bank}
        if path == '/transfers' and method == 'POST':
            check('TransferRequest', body)
            identifier = body['interbankTransferId']
            if headers.get('Idempotency-Key') != identifier:
                raise ValueError('Idempotency-Key must equal interbankTransferId')
            if body['originBankCode'] != self.partner or body['destinationBankCode'] != self.bank:
                raise Fault(401, 'UNAUTHORIZED', 'Bank does not participate')
            if timestamp(body['expiresAt']).timestamp() - timestamp(body['createdAt']).timestamp() != 180:
                raise ValueError('expiresAt must be createdAt plus 180 seconds')
            canonical = json.dumps(body, sort_keys=True, separators=(',', ':'))
            row = self.row(identifier)
            if row:
                if row['body'] is not None and row['body'] != canonical:
                    raise Fault(409, 'IDEMPOTENCY_CONFLICT', 'Same id with different content')
                if row['body'] is None:
                    self.db.execute('UPDATE transfers SET body=?,received=?,trace=?,callback_due=? WHERE id=?',
                                    (canonical, utc(self.clock()), headers['traceparent'], self.clock(), identifier))
                return 200, self.status(row)
            if timestamp(body['expiresAt']).timestamp() <= self.clock():
                raise Fault(422, 'EXPIRED', 'Transfer already expired; not processed')
            if self.scenario == 'unavailable':
                raise Fault(503, 'SERVICE_UNAVAILABLE', 'Mock unavailable scenario')
            now = utc(self.clock())
            self.db.execute('INSERT INTO transfers(id,body,status,received,updated,due,trace) VALUES(?,?,\'PROCESSING\',?,?,?,?)',
                            (identifier, canonical, now, now, self.clock()+self.delay, headers['traceparent']))
            return 202, {'interbankTransferId': identifier, 'status': 'PROCESSING', 'receivedAt': now}
        match = re.fullmatch(r'/transfers/([^/]+)(/(cancel|result))?', path)
        if match:
            identifier, _, operation = match.groups()
            uuid.UUID(identifier)
            row = self.row(identifier)
            if operation == 'result' and method == 'POST':
                check('TransferResult', body)
                outgoing = self.db.execute('SELECT * FROM outgoing WHERE id=?', (identifier,)).fetchone()
                if not outgoing:
                    raise Fault(404, 'NOT_FOUND', 'Unknown outbound test fixture')
                if body['interbankTransferId'] != identifier or body['destinationBankCode'] != self.partner:
                    raise Fault(401, 'UNAUTHORIZED', 'Callback direction or id mismatch')
                previous = outgoing['result']
                # Retain the first result; never apply a second fictional movement.
                if previous is None:
                    self.db.execute('UPDATE outgoing SET result=? WHERE id=?', (json.dumps(body), identifier))
                return 200, {'interbankTransferId': identifier, 'received': True}
            if operation == 'cancel' and method == 'POST':
                check('CancelRequest', body)
                if body['originBankCode'] != self.partner:
                    raise Fault(401, 'UNAUTHORIZED', 'Bank does not participate')
                result = 'CANCELLED'
                if row and row['status'] == 'ACCEPTED':
                    result = 'ALREADY_ACCEPTED'
                elif row and row['status'] == 'REJECTED':
                    result = 'ALREADY_REJECTED'
                else:
                    self.db.execute('INSERT INTO transfers(id,status,reason,updated,callback_due) VALUES(?,\'CANCELLED\',\'CANCELLED\',?,?) ON CONFLICT(id) DO UPDATE SET status=\'CANCELLED\',reason=\'CANCELLED\',updated=excluded.updated,callback_due=excluded.callback_due',
                                    (identifier, utc(self.clock()), self.clock()))
                return 200, {'interbankTransferId': identifier, 'result': result}
            if not operation and method == 'GET':
                if not row:
                    raise Fault(404, 'NOT_FOUND', 'Unknown transfer')
                return 200, self.status(row)
        if path == '/reversals' and method == 'POST':
            check('ReversalRequest', body)
            original = self.row(body['originalTransferId'])
            if body['originBankCode'] != self.partner:
                raise Fault(401, 'UNAUTHORIZED', 'Bank does not participate')
            if not original or original['status'] != 'ACCEPTED':
                raise Fault(404, 'NOT_FOUND', 'No accepted original transfer')
            if body['amount'] != json.loads(original['body'])['amount']:
                raise ValueError('Reversal amount must match original')
            canonical = json.dumps(body, sort_keys=True)
            previous = self.db.execute('SELECT * FROM reversals WHERE id=?', (body['reversalId'],)).fetchone()
            if previous and previous['body'] != canonical:
                raise ValueError('Reversal id reused with different content')
            self.db.execute('INSERT OR IGNORE INTO reversals VALUES(?,?,\'PROCESSING\')', (body['reversalId'], canonical))
            return 202, {'reversalId': body['reversalId'], 'status': 'PROCESSING'}
        match = re.fullmatch(r'/reversals/([^/]+)', path)
        if match and method == 'GET':
            identifier = str(uuid.UUID(match[1]))
            row = self.db.execute('SELECT * FROM reversals WHERE id=?', (identifier,)).fetchone()
            if not row:
                raise Fault(404, 'NOT_FOUND', 'Unknown reversal')
            return 200, {'reversalId': identifier, 'status': row['status']}
        if path == '/reconciliation' and method == 'GET':
            from datetime import date
            day = date.fromisoformat(query['date'][0]).isoformat()
            page, size = int(query.get('page', ['1'])[0]), int(query.get('pageSize', ['100'])[0])
            if page < 1 or not 1 <= size <= 500:
                raise ValueError('Invalid pagination')
            rows = self.db.execute('SELECT * FROM transfers WHERE body IS NOT NULL AND substr(received,1,10)=? ORDER BY id', (day,)).fetchall()
            items = []
            for row in rows[(page-1)*size:page*size]:
                value = json.loads(row['body'])
                item = self.status(row)
                item.pop('reasonCode', None)
                items.append(dict(item, amount=value['amount'], currency=value['currency']))
            return 200, dict(date=day, items=items, page=page, pageSize=size, totalItems=len(rows))
        raise Fault(404, 'NOT_FOUND', 'Unknown contract route')

    def process(self):
        with self.lock, self.db:
            for row in self.db.execute('SELECT * FROM transfers WHERE status=\'PROCESSING\' AND due<=?', (self.clock(),)).fetchall():
                data = json.loads(row['body'])
                reason = None
                if timestamp(data['expiresAt']).timestamp() <= self.clock():
                    reason = 'EXPIRED'
                elif Decimal(data['amount']) < Decimal('0.01') or Decimal(data['amount']) > Decimal('10000.00'):
                    reason = 'AMOUNT_LIMIT_EXCEEDED'
                elif data['destinationAccount'] not in self.accounts:
                    reason = 'ACCOUNT_NOT_FOUND'
                elif self.scenario == 'reject':
                    reason = 'PROCESSING_ERROR'
                status = 'REJECTED' if reason else 'ACCEPTED'
                if status == 'ACCEPTED':
                    self.db.execute('INSERT OR IGNORE INTO credits VALUES(?,?,?)', (row['id'], data['destinationAccount'], data['amount']))
                self.db.execute('UPDATE transfers SET status=?,reason=?,updated=?,callback_due=? WHERE id=?',
                                (status, reason, utc(self.clock()), self.clock(), row['id']))
            callbacks = self.db.execute('SELECT * FROM transfers WHERE body IS NOT NULL AND callback_due<=?', (self.clock(),)).fetchall()
        if not self.callback_url or self.scenario == 'no_callback':
            return
        for row in callbacks:
            if self.clock() - timestamp(row['updated']).timestamp() > 86400:
                with self.lock, self.db:
                    self.db.execute('UPDATE transfers SET callback_due=NULL WHERE id=?', (row['id'],))
                continue
            body = dict(interbankTransferId=row['id'], destinationBankCode=self.bank,
                        status='ACCEPTED' if row['status'] == 'ACCEPTED' else 'REJECTED', processedAt=row['updated'])
            if row['reason']:
                body['reasonCode'] = row['reason']
            trace = row['trace'].split('-')
            headers = {'Content-Type': 'application/json', 'X-Api-Key': self.callback_key,
                       'traceparent': '-'.join([trace[0], trace[1], secrets.token_hex(8), trace[3]])}
            request = urllib.request.Request(self.callback_url+'/transfers/'+row['id']+'/result',
                                             headers=headers, data=json.dumps(body).encode())
            delivered = False
            try:
                # Never forward credentials to an HTTP redirect target.
                class NoRedirect(urllib.request.HTTPRedirectHandler):
                    def redirect_request(self, *args, **kwargs):
                        return None
                with urllib.request.build_opener(NoRedirect).open(request, timeout=2) as response:
                    ack = json.loads(response.read(65536))
                    check('CallbackAck', ack)
                    delivered = response.status == 200 and ack['received'] is True and ack['interbankTransferId'] == row['id']
            except Exception:
                pass  # Keys, response bodies and connection details are never logged.
            with self.lock, self.db:
                next_at = None if delivered else self.clock()+min(300, 2**min(row['attempts']+1, 9))
                self.db.execute('UPDATE transfers SET callback_due=?,attempts=attempts+1 WHERE id=?', (next_at, row['id']))


class Server(ThreadingHTTPServer):
    daemon_threads = True

    def __init__(self, state, api_key, port=0):
        if len(api_key) < 16:
            raise ValueError('Provide MOCK_API_KEY of at least 16 characters')
        self.state, self.api_key = state, api_key
        super().__init__(('127.0.0.1', port), Handler)


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_GET(self):
        self.handle_request()

    def do_POST(self):
        self.handle_request()

    def handle_request(self):
        parsed = urlsplit(self.path)
        path = parsed.path.removeprefix('/interbank/v1')
        contract_path = next((p for p in CONTRACT['paths'] if re.fullmatch(re.sub(r'\{[^}]+\}', '[^/]+', p), path)), None)
        try:
            if not parsed.path.startswith('/interbank/v1/') or contract_path is None:
                raise Fault(404, 'NOT_FOUND', 'Unknown contract route')
            if not (path == '/health' and self.command == 'GET'):
                if not hmac.compare_digest(self.headers.get('X-Api-Key', ''), self.server.api_key):
                    raise Fault(401, 'UNAUTHORIZED', 'Invalid API key')
                validate(CONTRACT['components']['parameters']['Traceparent']['schema'], self.headers.get('traceparent'))
                trace = self.headers['traceparent'].split('-')
                if trace[1] == '0'*32 or trace[2] == '0'*16:
                    raise ValueError('Zero trace or span id')
            body = None
            if self.command == 'POST':
                if self.headers.get_content_type() != 'application/json':
                    raise ValueError('application/json required')
                length = int(self.headers.get('Content-Length', '0'))
                if not 0 < length <= 65536:
                    raise ValueError('Invalid request size')
                body = json.loads(self.rfile.read(length))
            code, value = self.server.state.dispatch(self.command, path, body, self.headers, parse_qs(parsed.query))
            validate(response_schema(contract_path, self.command, code), value)
        except Fault as error:
            code, value = error.status, dict(code=error.code, message=error.message, timestamp=utc())
        except (ValueError, KeyError, TypeError):
            code, value = 400, dict(code='VALIDATION_ERROR', message='Request does not satisfy the contract', timestamp=utc())
        data = json.dumps(value).encode()
        self.send_response(code)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(data)))
        self.end_headers()
        self.wfile.write(data)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--port', type=int, default=8099)
    parser.add_argument('--bank', default='BANK14')
    parser.add_argument('--partner', default='BANK03')
    parser.add_argument('--database', default='tools/interbank_mock/.state/mock.sqlite3')
    parser.add_argument('--delay', type=float, default=0.1)
    parser.add_argument('--scenario', choices=['accept', 'reject', 'no_callback', 'unavailable'], default='accept')
    parser.add_argument('--cert'); parser.add_argument('--key')
    args = parser.parse_args()
    if bool(args.cert) != bool(args.key):
        parser.error('Provide both certificate and private key')
    Path(args.database).parent.mkdir(parents=True, exist_ok=True)
    state = State(args.database, bank=args.bank, partner=args.partner,
                  accounts=os.environ.get('MOCK_ACCOUNTS', 'TEST-BANK14-001').split(','),
                  delay=args.delay, scenario=args.scenario, callback_url=os.environ.get('MOCK_PARTNER_BASE_URL', ''),
                  callback_key=os.environ.get('MOCK_PARTNER_API_KEY', ''))
    server = Server(state, os.environ.get('MOCK_API_KEY', ''), args.port)
    if args.cert:
        context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        context.load_cert_chain(args.cert, args.key)
        server.socket = context.wrap_socket(server.socket, server_side=True)
    stop = threading.Event()
    def worker():
        while not stop.wait(0.1):
            state.process()
    thread = threading.Thread(target=worker, daemon=True); thread.start()
    print('Local contract mock ready on loopback. No real bank or funds involved.', flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close(); stop.set(); thread.join(); state.db.close()


if __name__ == '__main__':
    main()
