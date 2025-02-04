from locust import HttpUser, task, between
import random
import json

class TokenUser(HttpUser):
    wait_time = between(0.001, 0.01)  # Simulating the time between requests to achieve high RPS

    @task
    def get_token(self):
        # Simulate a login request to the /token endpoint
        payload = {
            "username": "testuser",  # Replace with an actual username
            "password": "testpassword"   # Replace with an actual password
        }
        headers = {
            "Content-Type": "application/json"
        }

        self.client.post("/token", json=payload, headers=headers)

    # If you want to simulate more complex behavior, like adding delays or custom user behavior,
    # you can define more tasks and override other lifecycle methods in this class.