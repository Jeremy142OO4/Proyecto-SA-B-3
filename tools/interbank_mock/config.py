"""Agreement validation for future real integration; not enabled in the Gateway."""
from dataclasses import dataclass, field
import os
import re
from urllib.parse import urlsplit


def base_url(value, allow_loopback_http=False):
    parsed = urlsplit(value)
    local = allow_loopback_http and parsed.scheme == 'http' and parsed.hostname in ('127.0.0.1', 'localhost', '::1')
    if not (parsed.scheme == 'https' or local) or not parsed.hostname or parsed.username or parsed.password or parsed.query or parsed.fragment:
        raise ValueError('A valid HTTPS base URL without credentials, query or fragment is required')
    if parsed.path.rstrip('/') != '/interbank/v1' or any(c in value for c in '<>{}'):
        raise ValueError('Base URL must end in /interbank/v1 and contain no placeholders')
    return value.rstrip('/')


@dataclass(frozen=True)
class BankSettings:
    bank_code: str
    partner_code: str
    public_base_url: str
    partner_base_url: str
    api_key: str = field(repr=False)
    partner_api_key: str = field(repr=False)

    @classmethod
    def from_env(cls, values=None):
        values = os.environ if values is None else values
        bank = values.get('INTERBANK_BANK_CODE', 'BANK03')
        partner = values.get('INTERBANK_PARTNER_BANK_CODE', 'BANK14')
        if not re.fullmatch(r'BANK[0-9]{2}', bank) or not re.fullmatch(r'BANK[0-9]{2}', partner) or bank == partner:
            raise ValueError('Distinct BANKxx identifiers required')
        keys = [values.get(k, '') for k in ('INTERBANK_API_KEY', 'INTERBANK_PARTNER_API_KEY')]
        if any(len(k) < 16 or k.startswith('CHANGE_ME') for k in keys):
            raise ValueError('Both API keys must come from private configuration')
        return cls(bank, partner, base_url(values.get('INTERBANK_PUBLIC_BASE_URL', '')),
                   base_url(values.get('INTERBANK_PARTNER_BASE_URL', '')), *keys)
