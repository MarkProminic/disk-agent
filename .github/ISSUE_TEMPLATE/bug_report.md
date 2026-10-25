---
name: Bug report
about: Create a report to help us improve
title: '[BUG] '
labels: 'bug'
assignees: ''
---

## Bug Description

A clear and concise description of what the bug is.

## Environment

**System Information:**

- OS: [e.g., Debian 13, OmniOS r151059]
- Disk Agent Version: [e.g., 0.1.0]
- Installation Method: [package/source]
- smartmontools Version: [e.g., 7.5]

**Configuration:**

- Port: [e.g., 9430]
- Bind Address: [e.g., 0.0.0.0]

## Steps to Reproduce

Steps to reproduce the behavior:

1. Go to '...'
2. Run '...'
3. See error

## Expected Behavior

A clear and concise description of what you expected to happen.

## Actual Behavior

A clear and concise description of what actually happened.

## Error Messages / Logs

If applicable, add error messages or log excerpts (the agent log lives at `<config dir>/logs/agent.log`):

```text
Paste error messages here
```

## API Requests (if applicable)

**Request:**

```bash
curl "http://127.0.0.1:9430/api/status"
```

**Response:**

```json
{
  "error": "Error details here"
}
```

## Additional Context

Add any other context about the problem here.

## Impact Assessment

- [ ] Critical (system unusable, security issue)
- [ ] High (major functionality broken)
- [ ] Medium (functionality impaired)
- [ ] Low (minor issue, workaround available)

**Affected Functionality:**

- [ ] Disk collection
- [ ] Pool collection
- [ ] API endpoints
- [ ] Authentication
- [ ] Configuration
- [ ] Logging
- [ ] Packaging
- [ ] Documentation

## Resource Understanding

I understand that this project is maintained with limited development resources and that:

- Response times may vary based on current workload and severity
- Critical and high-impact issues receive priority attention
- Detailed bug reports help prioritize and resolve issues more efficiently
