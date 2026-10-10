# Security policy

## Supported versions

Security fixes are applied to the latest release and the `master` branch.

## Reporting a vulnerability

Please use GitHub's private vulnerability reporting for this repository. Do
not open a public issue for an undisclosed vulnerability. Include a minimal
reproducer, the affected revision, impact, and any suggested mitigation.

## Trust boundary

`php53` is a compatibility interpreter, not a sandbox. Interpreted programs
can access the filesystem and execute shell commands through PHP backticks.
Only run trusted PHP source, or isolate the process with operating-system or
container controls. Do not expose the interpreter directly to untrusted web
requests.
