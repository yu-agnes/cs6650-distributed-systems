"""
HW8 Performance Test Script
Runs 150 operations: 50 create cart, 50 add items, 50 get cart
Usage:
  python3 test_performance.py http://ALB_DNS_NAME mysql_test_results.json
  python3 test_performance.py http://ALB_DNS_NAME dynamodb_test_results.json
"""

import sys
import json
import time
import random
import urllib.request
import urllib.error
from datetime import datetime, timezone


def make_request(method, url, body=None):
    """Send HTTP request and return (status_code, response_body, elapsed_ms)"""
    headers = {"Content-Type": "application/json"} if body else {}
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, headers=headers, method=method)

    start = time.time()
    try:
        with urllib.request.urlopen(req) as resp:
            elapsed = (time.time() - start) * 1000  # ms
            body_text = resp.read().decode()
            return resp.status, body_text, elapsed
    except urllib.error.HTTPError as e:
        elapsed = (time.time() - start) * 1000
        return e.code, "", elapsed
    except Exception as e:
        elapsed = (time.time() - start) * 1000
        print(f"  ERROR: {e}")
        return 0, "", elapsed


def run_test(base_url, output_file):
    results = []
    cart_ids = []

    print(f"Target: {base_url}")
    print(f"Output: {output_file}")
    print()

    # ==================== Phase 1: Create 50 carts ====================
    print("Phase 1: Creating 50 carts...")
    for i in range(50):
        customer_id = random.randint(1, 1000)
        status, body, elapsed = make_request(
            "POST",
            f"{base_url}/shopping-carts",
            {"customer_id": customer_id}
        )

        success = status == 201
        if success and body:
            try:
                cart_id = json.loads(body)["shopping_cart_id"]
                cart_ids.append(cart_id)
            except (json.JSONDecodeError, KeyError):
                success = False

        results.append({
            "operation": "create_cart",
            "response_time": round(elapsed, 1),
            "success": success,
            "status_code": status,
            "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        })

        if (i + 1) % 10 == 0:
            print(f"  {i + 1}/50 done")

    print(f"  Created {len(cart_ids)} carts successfully")

    # Brief pause to allow eventual consistency to settle
    print("  Waiting 2 seconds for consistency...")
    time.sleep(2)
    print()

    # ==================== Phase 2: Add items to 50 carts ====================
    print("Phase 2: Adding items to 50 carts...")
    for i in range(50):
        cart_id = cart_ids[i % len(cart_ids)]  # cycle through available carts
        product_id = random.randint(1, 500)
        quantity = random.randint(1, 10)

        status, body, elapsed = make_request(
            "POST",
            f"{base_url}/shopping-carts/{cart_id}/items",
            {"product_id": product_id, "quantity": quantity}
        )

        results.append({
            "operation": "add_items",
            "response_time": round(elapsed, 1),
            "success": status == 204,
            "status_code": status,
            "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        })

        if (i + 1) % 10 == 0:
            print(f"  {i + 1}/50 done")

    # Brief pause to allow eventual consistency to settle
    print("  Waiting 2 seconds for consistency...")
    time.sleep(2)
    print()

    # ==================== Phase 3: Get 50 carts ====================
    print("Phase 3: Getting 50 carts...")
    for i in range(50):
        cart_id = cart_ids[i % len(cart_ids)]

        status, body, elapsed = make_request(
            "GET",
            f"{base_url}/shopping-carts/{cart_id}"
        )

        results.append({
            "operation": "get_cart",
            "response_time": round(elapsed, 1),
            "success": status == 200,
            "status_code": status,
            "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        })

        if (i + 1) % 10 == 0:
            print(f"  {i + 1}/50 done")

    print()

    # ==================== Save results ====================
    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)

    # ==================== Print summary ====================
    print("=" * 50)
    print("RESULTS SUMMARY")
    print("=" * 50)

    for op in ["create_cart", "add_items", "get_cart"]:
        op_results = [r for r in results if r["operation"] == op]
        times = [r["response_time"] for r in op_results]
        successes = sum(1 for r in op_results if r["success"])
        times.sort()

        print(f"\n{op}:")
        print(f"  Count:    {len(op_results)}")
        print(f"  Success:  {successes}/{len(op_results)}")
        print(f"  Avg:      {sum(times) / len(times):.1f} ms")
        print(f"  P50:      {times[len(times) // 2]:.1f} ms")
        print(f"  P95:      {times[int(len(times) * 0.95)]:.1f} ms")
        print(f"  P99:      {times[int(len(times) * 0.99)]:.1f} ms")

    all_times = [r["response_time"] for r in results]
    all_success = sum(1 for r in results if r["success"])
    all_times.sort()

    print(f"\nOVERALL:")
    print(f"  Total:    {len(results)}")
    print(f"  Success:  {all_success}/{len(results)} ({all_success / len(results) * 100:.1f}%)")
    print(f"  Avg:      {sum(all_times) / len(all_times):.1f} ms")
    print(f"  P50:      {all_times[len(all_times) // 2]:.1f} ms")
    print(f"  P95:      {all_times[int(len(all_times) * 0.95)]:.1f} ms")
    print(f"  P99:      {all_times[int(len(all_times) * 0.99)]:.1f} ms")

    print(f"\nResults saved to: {output_file}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 test_performance.py <base_url> <output_file>")
        print("Example: python3 test_performance.py http://hw8-alb-123456.us-east-1.elb.amazonaws.com mysql_test_results.json")
        sys.exit(1)

    run_test(sys.argv[1], sys.argv[2])
