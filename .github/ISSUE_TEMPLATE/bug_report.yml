name: Bug report
description: Report behavior that differs from what RouteBurn documents
labels: ["bug"]
body:
  - type: markdown
    attributes:
      value: |
        Thanks for the report. Please do not paste API keys, tokens, account-wide
        spend figures, or full HTTP responses. RouteBurn's own output is usually
        enough.
  - type: textarea
    id: what-happened
    attributes:
      label: What happened?
      description: Describe the actual behavior, including the exact message shown.
    validations:
      required: true
  - type: textarea
    id: expected
    attributes:
      label: What did you expect?
    validations:
      required: true
  - type: textarea
    id: steps
    attributes:
      label: Steps to reproduce
      placeholder: |
        1. run `routeburn`
        2. press `r`
        3. ...
    validations:
      required: true
  - type: input
    id: version
    attributes:
      label: RouteBurn version
      description: Output of `routeburn --version`.
    validations:
      required: true
  - type: input
    id: platform
    attributes:
      label: Platform
      description: OS, architecture, and terminal, for example macOS 15 on arm64 in iTerm2.
    validations:
      required: true
  - type: dropdown
    id: area
    attributes:
      label: Area
      options:
        - Dashboard rendering
        - Runway calculation
        - Charts or model breakdown
        - Refresh, cache, or offline behavior
        - Credentials or setup
        - Configuration or CLI flags
        - JSON mode
        - Release or installation
        - Other
    validations:
      required: true
  - type: textarea
    id: logs
    attributes:
      label: Relevant output
      description: Redacted stderr, or `routeburn --json` output with values rounded.
      render: shell
    validations:
      required: false
