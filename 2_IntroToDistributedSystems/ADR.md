---
status: "accepted"
date: 2025-02-04
decision-makers: ["Arslan Koshimov"]
consulted: []
informed: ["Project Owner"]
---

# OAuth Token Management System Implementation

## Context and Problem Statement

OAuth token management system should handle 1.000 users, each representing an external system. The system must issue tokens with a two-hour lifetime, refreshing them only when 75% of their lifetime has expired (for optimization). The solution may be implemented in three languages: Python, Golang, and Java. System must hold 20.000 RPS.

## Decision Drivers

* High request throughput (20,000 RPS).
* Efficient database connection management for scalability.

## Considered Options

* **Golang with Fiber and PostgreSQL**
* **Python with FastAPI and PostgreSQL**
* **Java with Spring Boot and PostgreSQL (-)**

## Decision Outcome

Chosen option: **Golang with Fiber and PostgreSQL**, since it provides superior performance compared to FastAPI in RPS. Given the high-load requirement, Go's lightweight concurrency model allows better utilization of system resources.

### Consequences

* **Good, because** Golang's performance under high concurrency significantly surpasses Python.

## Confirmation

To validate the implementation, we will:
* Use **Locust** for performance benchmarking.
* Compare the response times, error rates, and latency for single-instance and multi-instance setups.

## Pros and Cons of the Options

### Golang with Fiber and PostgreSQL

* **Good, because** Go offers high concurrency handling.
* **Good, because** Fiber is lightweight and provides minimal overhead.
* **Good, because** PostgreSQL is a well-supported, feature-rich database that fits well with Golang.

### Python with FastAPI and PostgreSQL

* **Good, because** FastAPI is developer-friendly.
* **Good, because** Python has a rich ecosystem and is easier to onboard developers.
* **Bad, because** Python struggles at RPS compared to Go.

### Java with Spring Boot and PostgreSQL

-

## Infrastructure Choices

### PostgreSQL with PgBouncer

* **Good, because** PostgreSQL is a robust and scalable relational database that supports ACID compliance.
* **Good, because** PgBouncer is lightweight connection pooler, reduce database connection overhead.
* **Bad, because** Some complex queries might introduce latency if not optimized properly.

### Nginx as a Reverse Proxy and Load Balancer

* **Good, because** Nginx efficiently handles high requests.
* **Good, because** It allows for rate-limiting, request logging, and caching.

## More Information

-
