[![Build and Test](https://github.com/wrouesnel/badgeserv/actions/workflows/integration.yml/badge.svg)](https://github.com/wrouesnel/badgeserv/actions/workflows/integration.yml)
[![Release](https://github.com/wrouesnel/badgeserv/actions/workflows/release.yml/badge.svg)](https://github.com/wrouesnel/badgeserv/actions/workflows/release.yml)
[![Container Build](https://github.com/wrouesnel/badgeserv/actions/workflows/container.yml/badge.svg)](https://github.com/wrouesnel/badgeserv/actions/workflows/container.yml)
[![Coverage Status](https://coveralls.io/repos/github/wrouesnel/badgeserv/badge.svg?branch=main)](https://coveralls.io/github/wrouesnel/badgeserv?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/wrouesnel/badgeserv)

# BadgeServ

No-Nonsense badge generator service. Ideal for on-premises hosting!

## Purpose

BadgeServ is designed to be completely un-opinionated about what sort of data you want to display, but to make displaying
any data easy. It includes no built-in badges or services at all and is intneded principally for supporting private
deployments which could benefit from a badge-generator service.

## Usage

### Swagger UI

All features can be explored from the swagger UI at `/api/v1/ui`

### Static Badges

`GET /api/v1/static?name=Name&value=Value&color=green`

Generate simple badges directly from a URL.

### Custom Badges

`GET /api/v1/dynamic/?target=https://my-json-service/this/should/be/encoded/properly&label=This can be Pongo2&message=So can this {{like.with.a.value}}`

Generate dynamic badges from any JSON endpoint using [pongo2](https://github.com/flosch/pongo2) for data
extraction.

Pongo2 is a Jinja2-like syntax derivative for Go, and is chosen because it provides advanced features like conditions
and text handling. Using this language in badge queries, almost any type of data can be handled.

## Coming Soon

The following features will be implemented soon

### Endpoint Badges

Endpoint badges implement a compatible interface similar to [shields.io](https://shields.io) and [badgen.net](https://badgen.net).

### Predefined Badges

`GET /api/v1/badge/<predefined name>/?param1=something&param2=something`

The predefined badge endpoints can be customized and configured when the service is deployed. This is a handy solution
for surfacing data which requires authentication tokens to retrieve. BadgeServ supports retrieving secrets from
Hashicorp Vault directly, for maximum configuration security.

### Web UI

A web UI beyond the default landing page.

## Acknowledgements

Adapted from original code by [Luzifer/badge-gen](https://github.com/Luzifer/badge-gen).
