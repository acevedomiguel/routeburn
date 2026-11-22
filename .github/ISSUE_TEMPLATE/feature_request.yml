name: Feature request
description: Suggest an improvement to RouteBurn
labels: ["enhancement"]
body:
  - type: markdown
    attributes:
      value: |
        RouteBurn is intentionally small: one provider, one 30-day range, read-only.
        Requests outside that scope are still welcome, but say why the narrower
        design does not work for you.
  - type: textarea
    id: problem
    attributes:
      label: What problem are you hitting?
      description: Describe the situation, not the specific implementation.
    validations:
      required: true
  - type: textarea
    id: proposal
    attributes:
      label: Proposed solution
    validations:
      required: true
  - type: textarea
    id: alternatives
    attributes:
      label: Alternatives considered
      description: Including doing nothing, scripts, or another tool.
    validations:
      required: false
  - type: dropdown
    id: scope
    attributes:
      label: Scope
      options:
        - Rendering or layout
        - Runway or analytics math
        - Configuration or CLI
        - Credentials
        - JSON contract
        - Release or packaging
        - Documentation
        - New provider support (out of MVP scope)
    validations:
      required: true
  - type: checkboxes
    id: constraints
    attributes:
      label: Constraints
      options:
        - label: This keeps RouteBurn read-only against OpenRouter.
        - label: This does not require storing secrets in plain text.
        - label: I understand the MVP supports only a 30-day range.