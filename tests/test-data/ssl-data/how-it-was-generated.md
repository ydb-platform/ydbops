1. Invoke this and interactively specify everything blank, but fqdn as "localhost"
- (specify any PEM phrase)

```
openssl req -new -addext "subjectAltName = DNS:localhost" -x509 -newkey rsa:4096 -days 365 -keyout ca.key -out ca.crt
openssl req -nodes -new -addext "subjectAltName = DNS:localhost" -x509 -subj "/C=AU/ST=Some-State/O=Internet Widgits Pty Ltd/CN=localhost" -newkey rsa:4096 -days 3650 -keyout ca.key -out ca.crt
```

2. Decrypt the cert by specifying the same PEM phrase
(maybe you can generate unencrypted in the first place, but in limited time I couldn't figure out how).
```
openssl rsa -in ca.key -out ca_unencrypted.key
```
