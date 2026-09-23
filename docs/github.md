# Publishing on GitHub

1. Unpack the archive; open a terminal inside `logicore-wms`.
2. Run the setup/demo and read `docs/verification.md`.
3. Create an **empty** GitHub repository named `logicore-wms` in your account.
4. From the project directory, run:

```bash
git init -b main
git add .
git commit -m "Build event-driven warehouse management backend"
git remote add origin https://github.com/YOUR_USERNAME/logicore-wms.git
git push -u origin main
```

Replace `YOUR_USERNAME` with your account. The Go module path is an internal monorepo import namespace and builds from the checkout; it does not require that GitHub organization to exist. If renaming the module, update all Go imports together.

Suggested description:

> Event-driven warehouse management backend: Java 21, Go, Kafka, PostgreSQL, transactional outbox, idempotency and concurrent inventory control.

Suggested topics: `java`, `golang`, `spring-boot`, `microservices`, `kafka`, `postgresql`, `warehouse-management`, `event-driven`, `docker-compose`, `testcontainers`.

Enable GitHub Actions and inspect the first `build-test-demo` run. Fix failures before claiming CI is green. After a passing run, add the repository-specific workflow badge if desired. Upload screenshots/recordings only from an actual run. Never commit `.env`, database dumps or build output.

No remote repository is created by unpacking this archive.
