"""
Experiment 1: Worker Scaling vs Throughput
Submit 500 independent jobs (1 URL each), measure total completion time.
Run with different worker_count (1, 4, 8) in Terraform.

Usage (Web UI):
  locust -f locustfile.py --host http://<ALB_DNS>
  Then open http://localhost:8089, set users=1, ramp up=1, start.

Usage (headless):
  locust -f locustfile.py --host http://<ALB_DNS> --headless -u 1 -r 1 --run-time 10m
"""

import json
import time
from locust import FastHttpUser, task, between


class ScrapeJobUser(FastHttpUser):
    wait_time = between(1, 2)

    @task
    def submit_batch_and_poll(self):
        num_urls = 500
        poll_interval = 3
        timeout = 600  # 10 min max

        # Phase 1: Submit 500 independent jobs (1 URL each)
        print(f"\n--- Submitting {num_urls} jobs ---")
        start_time = time.time()
        job_ids = []

        for i in range(num_urls):
            with self.client.post(
                "/jobs",
                json={"urls": [f"page-{i}"]},
                catch_response=True,
                name="POST /jobs"
            ) as response:
                if response.status_code == 202:
                    job_id = response.json().get("jobID")
                    if job_id:
                        job_ids.append(job_id)
                else:
                    response.failure(f"Expected 202, got {response.status_code}")

        submit_time = time.time() - start_time
        print(f"Submitted {len(job_ids)} jobs in {submit_time:.1f}s")

        # Phase 2: Poll all jobs until completed
        pending = set(job_ids)
        completed_count = 0

        while pending and (time.time() - start_time) < timeout:
            time.sleep(poll_interval)

            # Check a batch of pending jobs
            still_pending = set()
            for job_id in pending:
                with self.client.get(
                    f"/jobs/{job_id}",
                    catch_response=True,
                    name="GET /jobs/{id}"
                ) as response:
                    if response.status_code == 200:
                        status = response.json().get("status")
                        if status == "completed":
                            completed_count += 1
                        elif status == "failed":
                            pass  # don't re-poll failed jobs
                        else:
                            still_pending.add(job_id)
                    else:
                        still_pending.add(job_id)

            pending = still_pending

            elapsed = time.time() - start_time
            print(f"  Progress: {completed_count}/{len(job_ids)} completed, "
                  f"{len(pending)} pending, {elapsed:.0f}s elapsed")

        # Final summary
        total_time = time.time() - start_time
        throughput = completed_count / total_time if total_time > 0 else 0
        print(f"\n=== RESULT: {completed_count}/{len(job_ids)} jobs completed "
              f"in {total_time:.1f}s ({throughput:.1f} URLs/sec) ===\n")
