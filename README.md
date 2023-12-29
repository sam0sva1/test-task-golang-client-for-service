## Test task

Required to implement a client for an external service that processes a limited number of items in batches over a limited period of time.
This repo contains workspace with:

- batchservice (target service with test-only implementation);
- batchclient (implemented client wrapper);
- test-app (server app that uses both batchservice and batchclient).


> Didn't provide configuration settings and graceful shutdown

---

### Every module contains Makefile commands
```
// Find results in "reports" directory of each module

- make run_linter
- make run_tests
- make show_coverage
```

### [Makefile for test-app](test-app/Makefile) also contains:
```
make run_dev_api
```
