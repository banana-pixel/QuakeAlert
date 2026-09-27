# Security Policy

QuakeAlert is a life-safety project. Security reports are taken seriously and
handled with priority.

## Reporting a vulnerability

Please report suspected vulnerabilities **privately**, not through public issues
or pull requests, so a fix can ship before details are public.

Preferred channel:

1. GitHub private vulnerability reporting on the repository
   (`https://github.com/banana-pixel/QuakeAlert`): open the **Security** tab and
   choose **Report a vulnerability**.
2. If that is unavailable, email the maintainer at
   `wiratara006@gmail.com` with the subject line `QuakeAlert Security`.

Please include:

- a description of the issue and its impact,
- the affected component (firmware, server, Android, contracts, or deploy),
- steps to reproduce or a proof of concept,
- any suggested remediation.

## What to expect

- Acknowledgement of your report as soon as the maintainer is able.
- An honest assessment of whether it is in scope and how it will be handled.
- Coordinated disclosure: please allow a reasonable window for a fix before
  publishing details.

## Scope

In scope: the firmware, server, Android client, contracts, and deployment
configuration in this repository.

Out of scope: third-party services (for example Firebase Cloud Messaging or the
hosting provider), and social-engineering or denial-of-service testing against
the live deployment. Do not test against production infrastructure without
explicit written permission from the maintainer.

## Handling of secrets

All credentials (database password, AES master key, JWT secret, admin API key,
MQTT passwords, per-node HMAC keys) are provided through the environment and are
never committed. If you believe a secret has been exposed, report it privately
through the channel above so it can be rotated.
