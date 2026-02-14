"""
Locust load test using HttpUser (standard Python requests library)
Run: locust -f locustfile_http.py
"""

from locust import HttpUser, task, between
import random
import threading


class ProductAPIHttpUser(HttpUser):
    """
    Standard HttpUser - uses Python requests library.
    Good for general testing and debugging.
    """
    
    wait_time = between(1, 3)
    
    # Thread-safe counter
    counter_lock = threading.Lock()
    product_counter = 0
    
    def get_next_product_id(self):
        with ProductAPIHttpUser.counter_lock:
            ProductAPIHttpUser.product_counter += 1
            return ProductAPIHttpUser.product_counter % 1000 + 1
    
    @task(1)
    def health_check(self):
        """GET /health"""
        self.client.get("/health")
    
    @task(3)
    def get_product(self):
        """GET /products/{id} - weighted 3x (read-heavy workload)"""
        product_id = random.randint(1, 100)
        with self.client.get(
            f"/products/{product_id}", 
            name="/products/[id]",
            catch_response=True
        ) as response:
            if response.status_code in [200, 404]:
                response.success()
    
    @task(1)
    def create_product(self):
        """POST /products/{id}/details"""
        product_id = self.get_next_product_id()
        
        payload = {
            "product_id": product_id,
            "sku": f"SKU-{product_id:05d}",
            "manufacturer": "Test Manufacturer",
            "category_id": random.randint(1, 10),
            "weight": random.randint(100, 5000),
            "some_other_id": random.randint(1, 100)
        }
        
        self.client.post(
            f"/products/{product_id}/details",
            json=payload,
            name="/products/[id]/details"
        )
    
    @task(1)
    def get_nonexistent_product(self):
        """GET /products/{id} - expect 404"""
        product_id = random.randint(100000, 999999)
        with self.client.get(
            f"/products/{product_id}", 
            name="/products/[id] (404)",
            catch_response=True
        ) as response:
            if response.status_code == 404:
                response.success()
