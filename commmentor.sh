#!/usr/bin/env bash
set -euo pipefail

COMMAND="${1:-}"
INPUT_FILE="${2:-}"
EXIT_CODE="${3:-}"

# Validate inputs
if [[ -z "$COMMAND" || -z "$INPUT_FILE" || -z "$EXIT_CODE" ]]; then
  echo "Usage: $0 <command> <input_file> <exit_code>"
  exit 1
fi

PR_NUMBER=$(jq -r ".pull_request.number" "$GITHUB_EVENT_PATH")
if [[ "$PR_NUMBER" == "null" ]]; then
  echo "Not a PR, skipping comment post."
  exit 0
fi

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "GITHUB_TOKEN environment variable missing."
  exit 1
fi

# Prepare environment
INPUT=$(cat "$INPUT_FILE" | sed 's/\x1b\[[0-9;]*m//g')
WORKSPACE="${TF_WORKSPACE:-default}"
DETAILS_STATE=" open"

ACCEPT_HEADER="Accept: application/vnd.github.v3+json"
AUTH_HEADER="Authorization: token $GITHUB_TOKEN"
CONTENT_HEADER="Content-Type: application/json"

PR_COMMENTS_URL=$(jq -r ".pull_request.comments_url" "$GITHUB_EVENT_PATH")
PR_COMMENT_URI=$(jq -r ".repository.issue_comment_url" "$GITHUB_EVENT_PATH" | sed "s|{/number}||g")

# Delete old comment if exists
echo "Checking for existing $COMMAND comment..."
PR_COMMENT_ID=$(curl -sS -H "$AUTH_HEADER" -H "$ACCEPT_HEADER" -L "$PR_COMMENTS_URL" \
  | jq -r '[.[] | select(.body|test ("### Terraform `'"$COMMAND"'`")) | .id] | first')

if [[ "$PR_COMMENT_ID" && "$PR_COMMENT_ID" != "null" ]]; then
  echo "Deleting old comment ID: $PR_COMMENT_ID"
  PR_COMMENT_URL="$PR_COMMENT_URI/$PR_COMMENT_ID"
  curl -sS -X DELETE -H "$AUTH_HEADER" -H "$ACCEPT_HEADER" -L "$PR_COMMENT_URL" > /dev/null
fi

# Build new comment
if [[ "$COMMAND" == "plan" ]]; then
  if [[ "$EXIT_CODE" == "0" || "$EXIT_CODE" == "2" ]]; then
    CLEAN_PLAN=$(echo "$INPUT" | sed -r '/^(An execution plan has been generated and is shown below.|Terraform used the selected providers to generate the following execution|No changes. Infrastructure is up-to-date.|No changes. Your infrastructure matches the configuration.|Note: Objects have changed outside of Terraform)$/,$!d')
    CLEAN_PLAN=$(echo "$CLEAN_PLAN" | sed -r '/Plan: /q')
    CLEAN_PLAN=${CLEAN_PLAN::65300}
    CLEAN_PLAN=$(echo "$CLEAN_PLAN" | sed -r 's/^([[:blank:]]*)([-+~])/\2\1/g')
    CLEAN_PLAN=$(echo "$CLEAN_PLAN" | sed -r 's/^~/!/g')

    PR_COMMENT="### Terraform \`plan\` Succeeded for Workspace: \`$WORKSPACE\`
<details$DETAILS_STATE><summary>Show Output</summary>

\`\`\`diff
$CLEAN_PLAN
\`\`\`
</details>"
  else
    PR_COMMENT="### Terraform \`plan\` Failed for Workspace: \`$WORKSPACE\`
<details$DETAILS_STATE><summary>Show Output</summary>

\`\`\`
$INPUT
\`\`\`
</details>"
  fi
else
  PR_COMMENT="### Terraform \`${COMMAND}\` Result
<details$DETAILS_STATE><summary>Show Output</summary>

\`\`\`
$INPUT
\`\`\`
</details>"
fi

# Post new comment
PR_PAYLOAD=$(jq -n --arg body "$PR_COMMENT" '{body: $body}')

echo "Posting new comment..."
curl -sS -X POST -H "$AUTH_HEADER" -H "$ACCEPT_HEADER" -H "$CONTENT_HEADER" -d "$PR_PAYLOAD" -L "$PR_COMMENTS_URL" > /dev/null
