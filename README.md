

# BadgeServ

No-Nonsense badge generator service. Ideal for on-premises hosting!

## Purpose

BadgeServ is designed to be completely un-opinionated about what sort of data you want to display, but to make displaying
any data easy. It includes no built-in badges or services at all and is intneded principally for supporting private
deployments which could benefit from a badge-generator service.

## Usage

### Static Badges

`GET /static?name=Name&value=Value&color=green`

Generate simple badges directly from a URL.

### Custom Badges

`GET /dynamic/?target=https://my-json-service/this/should/be/encoded/properly&name=This can be Pongo2&value=So can this {{like.with.a.value}}`

Generate dynamic badges from any JSON or XML endpoint using [pongo2](https://github.com/flosch/pongo2) for data
extraction.

Pongo2 is a Jinja2-like syntax derivative for Go, and is chosen because it provides advanced features like conditions
and text handling. Using this language in badge queries, almost any type of data can be handled.

### Endpoint Badges

Endpoint badges implement a comaptible interface similar to [shields.io](https://shields.io) and [badgen.net](https://badgen.net).

### Predefined Badges

`GET /badge/<predefined name>/?param1=something&param2=something`

The predefined badge endpoints can be customized and configured when the service is deployed. This is a handy solution
for surfacing data which requires authentication tokens to retrieve. BadgeServ supports retrieving secrets from
Hashicorp Vault directly, for maximum configuration security.

## Acknowledgements

Adapted from original code by [Luzifer/badge-gen](https://github.com/Luzifer/badge-gen).
