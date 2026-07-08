# VaultPay API Engine

VaultPay is a high-throughput payment processing engine built from scratch in Go. The goal of this project was to design a resilient, containerized backend capable of handling financial transactions with absolute consistency. To achieve this, it uses a Redis distributed lock to prevent double-spending, a Master-Replica database setup for high-availability reads, and a guaranteed Transactional Outbox Pattern to ensure reliable event delivery to third-party webhooks.

---

## Architecture & Core Design Patterns

### 1. Idempotency & Race Condition Prevention (Redis)
To ensure a customer is never charged twice due to network retries or double-clicks, the system enforces strict idempotency using an `X-Idempotency-Key` HTTP header. Before processing a payment, the API attempts to secure an atomic distributed lock in Redis. Duplicate requests are instantly rejected before hitting the core database logic.

### 2. Reliable Webhook Delivery via the Transactional Outbox Pattern
Making external network requests directly during a live database transaction is risky because network drops can cause rollbacks or lock-ups. Instead, this engine writes payment events to an `outbox_events` table inside the exact same atomic transaction that updates user balances. 

A background poller runs every 2 seconds, pulling unprocessed entries safely using `FOR UPDATE SKIP LOCKED`, and passes them off to an Asynq/Redis distributed queue. If an external merchant webhook is down, the system retries gracefully without dropping data.

### 3. Read/Write Database Splitting
The infrastructure separates database concerns to prevent resource bottlenecks. Core application state modifications and financial balance entries are directed to a primary PostgreSQL **Master** instance, while read-only traffic (such as historical ledger checks or balance lookups) queries the **Replica** instance.

---

## Tech Stack

* **Language:** Go (Golang 1.26)
* **Databases:** PostgreSQL 16 (Master/Replica Replication Cluster)
* **Caching & Queue:** Redis 7 / Asynq Background Worker Engine
* **Containerization:** Docker & Docker Compose
* **Telemetry & Observability:** Prometheus & Grafana Engine
* **Documentation Interface:** Swagger UI (Open-API 3.0)

---

## 📊 High-Load Stress Testing (k6 Performance Results)

The engine was stress-tested using **Grafana k6** to simulate aggressive concurrent payment spikes, ramping up to **20 concurrent Virtual Users** blasting back-to-back charges. 

### Performance Benchmarks:
* **Total Transactions Dispatched:** 5,683 successful HTTP requests handled in 40 seconds.
* **Throughput Rate:** ~141.8 requests per second (RPS).
* **System Stability:** **100.00% Success Rate (0.00% failed)** across the entire runtime.
* **Ultra-Low Latency Profile:**
  * **Average Response Time (`avg`):** 5.15ms
  * **Median Response Time (`med`):** 4.35ms
  * **95th Percentile (`p95`):** 11.39ms *(Even under peak load, 95% of users received an API response in under 12 milliseconds!)*

The API gateway handled the traffic spike seamlessly, maintaining tight consistency across data-layer balances while background workers drained the generated webhook queues efficiently.

---

## Spin Up the Environment (Zero Dependencies)

The entire microservice setup is fully containerized, meaning you don't need Go, Postgres, or Redis installed on your local host machine to run it.

### 1. Check Prerequisites
Make sure your cryptographic signing keys are dropped directly into the root directory:
```bash
ls private_key.pem public_key.pem