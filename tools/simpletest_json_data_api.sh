#!/bin/bash
set -eu

API="http://localhost:1323/api/jsonData"

echo CREATE test-1
curl -X POST -H 'Content-Type: application/json' -d '{"data": {"number": 1, "string": "string", "array": [1, "string"], "object": {"key1": "value1"}}}' $API/test-1
echo GET test-1
curl -X GET $API/test-1
echo DELETE test-1
curl -X DELETE $API/test-1
echo GET test-1: not exists
curl -X GET $API/test-1
