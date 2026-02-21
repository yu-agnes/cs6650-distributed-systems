from locust import FastHttpUser, task
import random

# Common search terms that will match products
SEARCH_TERMS = [
    "alpha", "beta", "gamma", "delta", "epsilon",
    "electronics", "books", "home", "garden", "sports",
    "toys", "clothing", "food", "health", "automotive",
    "product", "zeta", "eta", "theta", "iota", "kappa"
]

class ProductSearchUser(FastHttpUser):
    # No wait time - maximum pressure to find bottlenecks

    @task(10)
    def search_products(self):
        term = random.choice(SEARCH_TERMS)
        self.client.get(f"/products/search?q={term}")

    @task(1)
    def health_check(self):
        self.client.get("/health")