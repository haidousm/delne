# Delne

Delne is Arabic for "show me the way". It is a simple reverse proxy with a web interface that allows you to easily show your requests the way ;)

## TO-DOs:

- [x] Support full path matching
- [x] Support subdomain matching
- [x] Support subpath matching & stripping
- [x] Ability add new services
- [x] Ability to delete services
- [x] Custom environment variables
- [ ] Ability to edit existing services' hosts
- [ ] Auto-gen SSL certs using Let's Encrypt (ACME)
    - [x] gen SSL certs for domains defined in `delne.toml`
    - [ ] gen SSL certs dynamically based on hosts inputted for services
- [ ] Add multi-host support
- [ ] Add authentication to the web interface
- [ ] Add ability to add custom headers to the request
- [ ] Add WS for real-time service status updates
- [ ] Add tests
- [ ] Update README with usage instructions
