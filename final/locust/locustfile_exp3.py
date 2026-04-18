"""
Experiment 3: Worker Failure and Job Recovery
Submits steady stream of jobs while you manually kill worker tasks.

Usage:
  locust -f locustfile_exp3.py --host http://<ALB_DNS>

Then open http://localhost:8089, set users=5, ramp up=5, start.

While running, open another terminal and kill a worker:
  aws ecs list-tasks --cluster scrape-pipeline-cluster --service-name scrape-pipeline-worker-service
  aws ecs stop-task --cluster scrape-pipeline-cluster --task <task-id>

Watch:
  - Locust Charts: RPS drop and recovery
  - Terminal: job completion tracking
  - AWS ECS Events: task stop/start times
  - AWS SQS Monitoring: messages becoming visible again after visibility timeout
"""

import time
import json
from locust import FastHttpUser, task, between, events


# Track job submissions and completions
submitted_jobs = []
completed_jobs = []
failed_jobs = []
lock = __import__('threading').Lock()


class FailureTestUser(FastHttpUser):
    wait_time = between(1, 2)

    @task(3)
    def submit_job(self):
        """Submit a single-URL job."""
        with self.client.post(
            "/jobs",
            json={"urls": [f"page-{int(time.time() * 1000) % 10000}"]},
            catch_response=True,
            name="POST /jobs"
        ) as response:
            if response.status_code == 202:
                job_id = response.json().get("jobID")
                if job_id:
                    with lock:
                        submitted_jobs.append({
                            "id": job_id,
                            "submitted_at": time.time()
                        })

    @task(1)
    def poll_jobs(self):
        """Poll oldest unfinished jobs to track completion."""
        with lock:
            # Get jobs that haven't been checked as completed yet
            to_check = [j for j in submitted_jobs if j["id"] not in
                       [c["id"] for c in completed_jobs] and
                       j["id"] not in [f["id"] for f in failed_jobs]]
            # Only check the 10 oldest to avoid flooding
            to_check = to_check[:10]

        for job in to_check:
            with self.client.get(
                f"/jobs/{job['id']}",
                catch_response=True,
                name="GET /jobs/{id}"
            ) as response:
                if response.status_code == 200:
                    data = response.json()
                    status = data.get("status")
                    if status == "completed":
                        with lock:
                            completed_jobs.append({
                                "id": job["id"],
                                "completed_at": time.time(),
                                "duration": time.time() - job["submitted_at"]
                            })
                    elif status == "failed":
                        with lock:
                            failed_jobs.append({
                                "id": job["id"],
                                "failed_at": time.time()
                            })


# Print summary every 15 seconds
start_time = None
last_print = 0

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    global start_time, last_print
    start_time = time.time()
    last_print = 0
    print("\n" + "=" * 60)
    print("EXPERIMENT 3: WORKER FAILURE AND JOB RECOVERY")
    print("=" * 60)
    print("Instructions:")
    print("  1. Start with 5 users, let it stabilize for 30 seconds")
    print("  2. Open another terminal and kill a worker task:")
    print("     aws ecs list-tasks --cluster scrape-pipeline-cluster \\")
    print("       --service-name scrape-pipeline-worker-service")
    print("     aws ecs stop-task --cluster scrape-pipeline-cluster \\")
    print("       --task <task-id>")
    print("  3. Watch this terminal for throughput drop and recovery")
    print("  4. After recovery, run for 2 more minutes then Stop")
    print("=" * 60 + "\n")


@events.request.add_listener
def on_request(request_type, name, response_time, response_length, **kwargs):
    global last_print, start_time
    if start_time is None:
        return

    elapsed = int(time.time() - start_time)

    if elapsed - last_print >= 15:
        last_print = elapsed
        with lock:
            total_submitted = len(submitted_jobs)
            total_completed = len(completed_jobs)
            total_failed = len(failed_jobs)
            pending = total_submitted - total_completed - total_failed

            # Calculate recent throughput (last 15 seconds)
            recent = [c for c in completed_jobs
                     if c["completed_at"] > time.time() - 15]
            recent_rate = len(recent) / 15.0

        print(f"  [{elapsed:>4d}s] Submitted: {total_submitted} | "
              f"Completed: {total_completed} | Failed: {total_failed} | "
              f"Pending: {pending} | Recent rate: {recent_rate:.1f} jobs/sec")


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    with lock:
        total = len(submitted_jobs)
        done = len(completed_jobs)
        fail = len(failed_jobs)
        lost = total - done - fail

    print("\n" + "=" * 60)
    print("EXPERIMENT 3: FINAL SUMMARY")
    print(f"  Total submitted:  {total}")
    print(f"  Completed:        {done}")
    print(f"  Failed:           {fail}")
    print(f"  Potentially lost: {lost}")
    print(f"  Loss rate:        {lost/total*100:.1f}%" if total > 0 else "  Loss rate: N/A")
    print("=" * 60 + "\n")
