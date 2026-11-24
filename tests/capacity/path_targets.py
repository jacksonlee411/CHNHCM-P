"""Approved Department path-filter stage target; retain the long-term goal."""

POLICY_ID = 'department-path-stage-1000ms-20261115'
CONTRACT = 'docs/specs/m02/120-dated-maintenance-technical-contract.md#department-path-capacity-target'
LONG_TERM_TARGET_MS = 300


def target_ms(*, sessions, long_term=False):
    # Explicit supported session counts prevent silently admitting a reduced
    # concurrency shape under the twenty-session target.
    return {1: 300, 20: 300 if long_term else 1000}[sessions]
