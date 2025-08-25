# ACME Client with Certificate Sharing API

## DESCRIPTION

A lightweight ACME client designed to automate the full lifecycle of SSL/TLS certificates.

It handles certificate issuance and renewal, and provides a secure HTTPS JSON API for distributing certificates to other servers.

## INSTALLATION

```bash
ln -sf /www/server/acme/acme.service /etc/systemd/system/acme.service
systemctl daemon-reload
systemctl enable --now acme.service
systemctl status acme.service
```
