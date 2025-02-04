from locust import HttpUser, task, between

class TokenLoadTest(HttpUser):
    wait_time = between(1, 10)  # Simulates user think time

    @task
    def get_access_token(self):
        with self.client.post(
            "/token",
            params={"username": "testuser", "password": "securepassword"},
            timeout=10,  # Matches Artillery's timeout setting
            catch_response=True
        ) as response:
            if response.status_code == 200:
                token = response.json().get("access_token")
                if token:
                    response.success()
                else:
                    response.failure("No access token in response")
            else:
                response.failure(f"Unexpected status code {response.status_code}")

# Locust expects a CLI-runner; define a config with CLI args:
# Example: `locust -f locustfile.py --host=http://localhost:8080 --users 1000 --spawn-rate 100 --run-time 60s`