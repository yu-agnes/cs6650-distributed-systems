"""
Experiment 2: Burst Absorption
Tests whether SQS successfully decouples API from workers under traffic spikes.

Phase 1 (baseline):  10 users, 30 seconds
Phase 2 (spike):    100 users, 2 minutes
Phase 3 (recovery):  10 users, 3 minutes

Usage:
  locust -f locustfile_exp2.py --host http://<ALB_DNS>

Then open http://localhost:8089
- Start with 10 users, ramp up 10
- After 30 seconds, click Edit and change to 100 users
- After 2 minutes at 100, click Edit and change back to 10 users
- Wait 3 minutes for queue to drain, then Stop

Watch the terminal for phase summaries.
"""

import time
import threading
from locust import FastHttpUser, task, between, events


class BurstTestUser(FastHttpUser):
    wait_time = between(0.5, 1.5)

    @task
    def submit_job(self):
        """Each user submits one job per cycle (1 URL each)."""
        with self.client.post(
            "/jobs",
            json={"urls": [f"page-{int(time.time() * 1000) % 10000}"]},
            catch_response=True,
            name="POST /jobs"
        ) as response:
            if response.status_code != 202:
                response.failure(f"Expected 202, got {response.status_code}")

    @task(3)
    def poll_random_job(self):
        """Poll a recent job to generate GET traffic and measure read latency."""
        # Use a dummy job ID - will return 404 but we measure API response time
        # In real scenario, we'd track submitted job IDs
        with self.client.post(
            "/jobs",
            json={"urls": [f"page-{int(time.time() * 1000) % 10000}"]},
            catch_response=True,
            name="POST /jobs (poll-setup)"
        ) as response:
            if response.status_code == 202:
                job_id = response.json().get("jobID", "")
                if job_id:
                    # Immediately poll - measures API + DynamoDB read latency
                    with self.client.get(
                        f"/jobs/{job_id}",
                        catch_response=True,
                        name="GET /jobs/{id}"
                    ) as poll_resp:
                        if poll_resp.status_code == 200:
                            poll_resp.success()


# Track and print stats periodically
start_time = None
last_print = 0

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    global start_time, last_print
    start_time = time.time()
    last_print = 0
    print("\n" + "=" * 60)
    print("EXPERIMENT 2: BURST ABSORPTION TEST")
    print("=" * 60)
    print("Instructions:")
    print("  1. Start with 10 users (baseline)")
    print("  2. After 30s, Edit -> change to 100 users (spike)")
    print("  3. After 2min at 100, Edit -> change to 10 users (recovery)")
    print("  4. Wait 3min for queue drain, then Stop")
    print("=" * 60 + "\n")


@events.request.add_listener
def on_request(request_type, name, response_time, response_length, **kwargs):
    global last_print, start_time
    if start_time is None:
        return

    elapsed = int(time.time() - start_time)

    # Print summary every 15 seconds
    if elapsed - last_print >= 15:
        last_print = elapsed

        if elapsed < 30:
            phase = "BASELINE (10 users)"
        elif elapsed < 150:
            phase = "SPIKE (100 users)"
        else:
            phase = "RECOVERY (10 users)"

        print(f"  [{elapsed:>4d}s] Phase: {phase}")
