"""Validate the schema subset actually used by the immutable course contract."""
from datetime import datetime, date
import hashlib
from pathlib import Path
import re
import uuid
import yaml

ROOT = Path(__file__).resolve().parents[2]
CONTRACT_PATH = ROOT / 'contracts/interbank/openapi.yaml'
CONTRACT_SHA256 = '4888b951aa48f95d8d301098c8aa642adfaa0e45a380975b8ce42acf3506df6a'
if hashlib.sha256(CONTRACT_PATH.read_bytes()).hexdigest() != CONTRACT_SHA256:
    raise ValueError('Original OpenAPI was modified; restore the course contract')
CONTRACT = yaml.safe_load(CONTRACT_PATH.read_text(encoding='utf-8'))


def resolve(schema):
    while '$ref' in schema:
        value = CONTRACT
        for key in schema['$ref'].removeprefix('#/').split('/'):
            value = value[key]
        schema = value
    return schema


def timestamp(value):
    parsed = datetime.fromisoformat(value.replace('Z', '+00:00'))
    if parsed.tzinfo is None or 'T' not in value:
        raise ValueError('Timezone required')
    return parsed


def validate(schema, value):
    schema = resolve(schema)
    kind = schema.get('type')
    valid = {'object': isinstance(value, dict), 'array': isinstance(value, list),
             'string': isinstance(value, str), 'integer': type(value) is int,
             'boolean': type(value) is bool}
    if kind and not valid.get(kind, False):
        raise ValueError('Invalid JSON type')
    if 'enum' in schema and value not in schema['enum']:
        raise ValueError('Value not in contract enum')
    if kind == 'object':
        if any(key not in value for key in schema.get('required', [])):
            raise ValueError('Missing required field')
        properties = schema.get('properties', {})
        if schema.get('additionalProperties') is False and set(value) - set(properties):
            raise ValueError('Unexpected field')
        for key in set(value) & set(properties):
            validate(properties[key], value[key])
    if kind == 'array':
        for item in value:
            validate(schema['items'], item)
    if kind == 'string':
        if len(value) > schema.get('maxLength', float('inf')):
            raise ValueError('String too long')
        if 'pattern' in schema and not re.fullmatch(schema['pattern'], value):
            raise ValueError('Invalid contract pattern')
        fmt = schema.get('format')
        if fmt == 'uuid' and str(uuid.UUID(value)) != value.lower():
            raise ValueError('Invalid UUID')
        if fmt == 'date-time':
            timestamp(value)
        if fmt == 'date':
            date.fromisoformat(value)
    if kind == 'integer' and not schema.get('minimum', -float('inf')) <= value <= schema.get('maximum', float('inf')):
        raise ValueError('Integer out of range')


def check(name, value):
    validate(CONTRACT['components']['schemas'][name], value)


def response_schema(path, method, status):
    result = resolve(CONTRACT['paths'][path][method.lower()]['responses'][str(status)])
    return result['content']['application/json']['schema']
