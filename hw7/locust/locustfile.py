"""
HW7 Load Testing - Sync vs Async Order Processing

=== Sync Tests ===
Normal:     locust -f locustfile.py SyncUser --host=http://YOUR-ALB -u 5 -r 1 -t 30s --headless
Flash Sale: locust -f locustfile.py SyncUser --host=http://YOUR-ALB -u 20 -r 10 -t 60s --headless

=== Async Tests ===
Flash Sale: locust -f locustfile.py AsyncUser --host=http://YOUR-ALB -u 20 -r 10 -t 60s --headless

=== Web UI (no --headless) ===
Sync:  locust -f locustfile.py SyncUser --host=http://YOUR-ALB
Async: locust -f locustfile.py AsyncUser --host=http://YOUR-ALB
"""

import random
from locust import FastHttpUser, task, between


def generate_order():
    """Generate a random order payload"""
    num_items = random.randint(1, 3)
    items = []
    for i in range(num_items):
        items.append({
            "product_id": random.randint(1, 1000),
            "name": f"Product-{random.randint(1, 100)}",
            "quantity": random.randint(1, 5),
            "price": round(random.uniform(9.99, 199.99), 2)
        })
    return {
        "customer_id": random.randint(1, 10000),
        "items": items
    }


class SyncUser(FastHttpUser):
    """Test synchronous order endpoint"""
    wait_time = between(0.1, 0.5)

    @task
    def place_order_sync(self):
        self.client.post(
            "/orders/sync",
            json=generate_order(),
            name="/orders/sync"
        )


class AsyncUser(FastHttpUser):
    """Test asynchronous order endpoint"""
    wait_time = between(0.1, 0.5)

    @task
    def place_order_async(self):
        self.client.post(
            "/orders/async",
            json=generate_order(),
            name="/orders/async"
        )
