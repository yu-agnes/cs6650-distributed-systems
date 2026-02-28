from locust import HttpUser, task, between
import random

search_terms = ["alpha", "beta", "gamma", "delta", "epsilon",
                "zeta", "eta", "theta", "iota", "kappa",
                "electronics", "books", "home", "garden", "sports",
                "toys", "food", "health", "product"]

class SearchUser(HttpUser):
    wait_time = between(0.1, 0.5)

    @task(5)
    def search_product(self):
        query = random.choice(search_terms)
        with self.client.get(
            f"/products/search?q={query}",
            name="/products/search",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Status {response.status_code}")

    @task(1)
    def health_check(self):
        with self.client.get(
            "/health",
            name="/health",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Status {response.status_code}")
