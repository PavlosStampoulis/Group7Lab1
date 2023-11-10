#!/bin/bash

TARGET_IP="3.91.39.148"
TARGET_PORT="1122"
PROXY_IP="54.80.84.65"
PROXY_PORT="1123"
OUTPUT_FILE="cat2.jpg"

for ((i = 1; i <= 11; i++)); do
    curl -X GET "${TARGET_IP}:${TARGET_PORT}/files/cat2.jpg" -x "${PROXY_IP}:${PROXY_PORT}" -o "${OUTPUT_FILE}_${i}.jpg"
done