#!/bin/bash
set -eu

API="http://localhost:1323/api/entities"
JSON='{"id":"test-1","type":"name","altLabels":["test-1a","test-1b"],"exactMatches":["uri-1a","uri-1b"]}'
JSONc='{"id":"test-1","type":"NAME","altLabels":["test-1a","test-1c"],"exactMatches":["uri-1a","uri-1c"]}'

echo GET all
RES="$(curl -s -X GET $API)"
EXPECTED="[]"
if [ "$RES" = "$EXPECTED" ]; then echo 'ok'; else echo "ng: $RES x $EXPECTED"; fi
echo CREATE test-1
RES="$(curl -s -X POST -H 'Content-Type: application/json' -d '{"id": "test-1", "type": "name", "altLabels": ["test-1a", "test-1b"], "exactMatches": ["uri-1a", "uri-1b"]}' $API)"
RES="$(echo "$RES" | sed -s 's/,"updated":"[^"]*"//g')"
EXPECTED="$JSON"
if [ "$RES" = "$EXPECTED" ]; then echo 'ok'; else echo "ng: $RES x $EXPECTED"; fi
echo GET all
RES="$(curl -s -X GET $API)"
RES="$(echo "$RES" | sed -s 's/,"updated":"[^"]*"//g')"
EXPECTED="[$JSON]"
if [ "$RES" = "$EXPECTED" ]; then echo 'ok'; else echo "ng: $RES x $EXPECTED"; fi
echo GET test-1
RES="$(curl -s -X GET $API/test-1)"
RES="$(echo "$RES" | sed -s 's/,"updated":"[^"]*"//g')"
EXPECTED="$JSON"
if [ "$RES" = "$EXPECTED" ]; then echo 'ok'; else echo "ng: $RES x $EXPECTED"; fi
echo DELETE test-1
RES="$(curl -s -X DELETE $API/test-1)"
EXPECTED='{"id":"test-1"}'
if [ "$RES" = "$EXPECTED" ]; then echo 'ok'; else echo "ng: $RES x $EXPECTED"; fi
echo GET all: none
RES="$(curl -s -X GET $API)"
EXPECTED="[]"
if [ "$RES" = "$EXPECTED" ]; then echo 'ok'; else echo "ng: $RES x $EXPECTED"; fi
echo GET test-1: error
RES="$(curl -s -X GET $API/test-1)"
EXPECTED='{"code":404,"error":"getentity","message":"test-1 not found"}'
if [ "$RES" = "$EXPECTED" ]; then echo 'ok'; else echo "ng: $RES x $EXPECTED"; fi
echo DELETE test-1: error
RES="$(curl -s -X DELETE $API/test-1)"
EXPECTED='{"code":404,"error":"deleteentity","message":"test-1 not found"}'
if [ "$RES" = "$EXPECTED" ]; then echo 'ok'; else echo "ng: $RES x $EXPECTED"; fi
