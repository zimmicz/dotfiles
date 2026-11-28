SHIPPED_ST='"On nightly", "To test", "Release candidate", "Rejected"'
PIPELINE_LABELS='triage, needs-answer, triaged, triage-failed, ai-fix, ai-mr, ai-fix-failed'
MR_STATES_JQ='[.[] | select(.title | test("(^|[^A-Z0-9-])" + $k + "([^0-9]|$)")) | .state] | unique | join(",")'
typeset -A PRODUCT=(cf-farm-frontend 11710 can-frontend 11707)
