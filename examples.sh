#!/bin/bash
# Example politest generate commands

# Bedrock - Invoke-only policy (least privilege for model invocation)
# With --scp flag, generates both identity policy AND Service Control Policy

# Allowed models (update this list as needed)
ALLOWED_MODELS=(
  "arn:aws:bedrock:eu-west-2::foundation-model/anthropic.claude-3-7-sonnet-20250219-v1:0"
)

MODELS_LIST=$(IFS=', '; echo "${ALLOWED_MODELS[*]}")
ALLOWED_REGION="eu-west-2"

go run . generate \
  --url "https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazonbedrock.html" \
  --base-url "https://api.openai.com" \
  --model "gpt-4o" \
  --api-key "$(cat ~/.ssh/openai_api_key)" \
  --scp \
  --prompt "I need a policy for a developer who can ONLY call specific Bedrock foundation models - nothing else.

ALLOWED MODELS: ${MODELS_LIST}
ALLOWED REGION: ${ALLOWED_REGION}

THE DEVELOPER SHOULD BE ABLE TO:
- Call the allowed models using InvokeModel and InvokeModelWithResponseStream
- Discover what models exist using Get*, Describe*, List* wildcards only (do NOT list individual Get/Describe/List actions)

THE DEVELOPER MUST NOT BE ABLE TO:
- Call any models other than those listed above
- Use cross-region inference or global cross-region inference (CRIS)
- Create, update, delete, or manage any Bedrock resources
- Invoke agents, flows, or anything other than foundation models
- Do anything administrative

POLICY SPLIT (SCP will be generated separately):
- Identity policy: Only ALLOW statements with actions and resources - no conditions like SecureTransport
- SCP: All security guardrails (SecureTransport, region locks, NotAction allowlist)

For the identity policy: Keep it minimal - just actions and resources. Do NOT include SecureTransport conditions or any deny statements - those all go in the SCP. Use ONLY wildcards for read operations - do not list individual actions that are already covered by wildcards."
