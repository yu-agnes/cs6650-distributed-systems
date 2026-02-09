# MapReduce Experiment Writeup

## Overview

For this assignment, I implemented a simplified MapReduce word count system using AWS ECS/Fargate and S3. The system consists of three components:

- **Splitter**: Splits the input file (Shakespeare's Hamlet, ~162KB) into 3 equal-sized chunks
- **Mapper**: Counts word occurrences in each chunk and outputs JSON results
- **Reducer**: Aggregates results from all mappers into a final word count


## Performance Experiment

I conducted two experiments to compare serial vs parallel execution:

### Experiment A: Serial Execution (1 Task)

All operations performed sequentially on a single ECS task:

| Phase | Time |
|-------|------|
| Split | 0.602s |
| Map chunk_0 | 0.295s |
| Map chunk_1 | 0.293s |
| Map chunk_2 | 0.317s |
| Reduce | 0.582s |
| **Total Map Time** | **0.905s** |
| **Total Time** | **2.089s** |

### Experiment B: Parallel Execution (5 Tasks)

Split and Reduce on dedicated tasks; 3 Mappers running in parallel:

| Phase | Time |
|-------|------|
| Split | 0.466s |
| Map (3 parallel) | 0.348s |
| Reduce | 0.439s |
| **Total Time** | **1.253s** |

## Results: Did I get a speedup?

**Yes!** The parallel execution showed significant improvement:

| Metric | Serial | Parallel | Speedup |
|--------|--------|----------|---------|
| Map Phase | 0.905s | 0.348s | **2.60x faster** |
| Total Time | 2.089s | 1.253s | **1.67x faster** |

The Map phase showed great improvement (2.60x) because all three mappers ran simultaneously on different ECS tasks instead of sequentially. The total speedup (1.67x) is less than 3x because Split and Reduce operations cannot be parallelized and still run sequentially.

## Final Results

The MapReduce system successfully processed Hamlet and produced correct results:

- **Total unique words**: 4,699
- **Final output**: `results/final_result.json` in S3


## Key Learnings

1. **Parallelism benefits**: Running 3 mappers in parallel reduced the map phase time by ~2.6x
2. **Coordination challenges**: Manually coordinating 5 tasks requires careful orchestration becuase we need to make sure that Split completes before Map, and all Maps complete before Reduce
3. **S3 as shared storage**: S3 served as the communication medium between tasks, which eliminates the need for direct task-to-task communication
