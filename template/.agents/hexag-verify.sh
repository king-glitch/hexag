#!/bin/sh
# Antigravity hook. PostToolUse output never reaches the model, so verify
# failures are fed back through PreInvocation (mid-task) and Stop (finish gate).
cd "$(dirname "$0")/.." || exit 0
conversation=$(sed -n 's/.*"conversationId": *"\([^"]*\)".*/\1/p')
state="${TMPDIR:-/tmp}/hexag-verify-stop-$conversation"

if make -s verify >/dev/null 2>&1; then
  rm -f "$state"
  if [ "$1" = stop ]; then echo '{"decision":"allow"}'; else echo '{}'; fi
  exit 0
fi

if [ "$1" = pre ]; then
  echo '{"injectSteps":[{"ephemeralMessage":"make verify is failing. Run make verify and fix every violation before the next edit (AGENTS.md Operating Loop)."}]}'
  exit 0
fi

# ponytail: 3 forced continues per failing streak, so an unfixable violation cannot loop forever.
attempts=$(($(cat "$state" 2>/dev/null || echo 0) + 1))
echo "$attempts" > "$state"
if [ "$attempts" -le 3 ]; then
  echo '{"decision":"continue","reason":"make verify is failing. Run make verify, fix every violation, then finish."}'
else
  echo '{"decision":"allow"}'
fi
