#!/bin/bash
set -eu

API="http://localhost:1323/api/jsonData"

echo GET all
curl -X GET $API
echo CREATE test-1
curl -X POST -H 'Content-Type: application/json' -d '{"key": "test-1", "data": {"number": 1, "string": "string", "array": [1, "string"], "object": {"key1": "value1"}}}' $API
echo GET all
curl -X GET $API
echo GET test-1
curl -X GET $API/test-1
echo DELETE test-1
curl -X DELETE $API/test-1
echo GET all: none
curl -X GET $API
echo GET test-1: error
curl -X GET $API/test-1
echo DELETE test-1: error
curl -X DELETE $API/test-1
