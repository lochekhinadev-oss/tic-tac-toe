#!/bin/sh
set -eu

if [ -n "${TELEGRAM_BOT_TOKEN:-}" ] && [ -n "${TELEGRAM_CHAT_ID:-}" ]; then
  cat <<EOF
global: {}

route:
  receiver: telegram
  group_by:
    - alertname
    - service
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

receivers:
  - name: telegram
    telegram_configs:
      - bot_token: "${TELEGRAM_BOT_TOKEN}"
        chat_id: ${TELEGRAM_CHAT_ID}
        send_resolved: true
        message: |-
          [{{ .Status | toUpper }}] {{ .CommonLabels.alertname }}
          Service: {{ .CommonLabels.service }}
          Summary: {{ .CommonAnnotations.summary }}
          Description: {{ .CommonAnnotations.description }}
EOF
  exit 0
fi

cat <<EOF
global: {}

route:
  receiver: blackhole
  group_by:
    - alertname
    - service
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

receivers:
  - name: blackhole
EOF
