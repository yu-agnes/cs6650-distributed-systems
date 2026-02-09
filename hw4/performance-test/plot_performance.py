import matplotlib.pyplot as plt
import numpy as np

# Performance data from experiments
# Serial test (1 Task) - all operations on same machine
serial_data = {
    'Split': 0.602,
    'Map (total)': 0.905,  # 0.295 + 0.293 + 0.317
    'Reduce': 0.582
}

# Parallel test (5 Tasks) - 3 mappers running simultaneously
parallel_data = {
    'Split': 0.466,
    'Map (total)': 0.348,  # 3 mappers in parallel
    'Reduce': 0.439
}

# Calculate totals
serial_total = sum(serial_data.values())
parallel_total = sum(parallel_data.values())

# Create figure with 2 subplots
fig, axes = plt.subplots(1, 2, figsize=(14, 6))

# ============ Plot 1: Bar chart comparing each phase ============
ax1 = axes[0]
phases = list(serial_data.keys())
x = np.arange(len(phases))
width = 0.35

bars1 = ax1.bar(x - width/2, serial_data.values(), width, label='Serial (1 Task)', color='#ff7f7f')
bars2 = ax1.bar(x + width/2, parallel_data.values(), width, label='Parallel (5 Tasks)', color='#7fbf7f')

ax1.set_xlabel('Phase', fontsize=12)
ax1.set_ylabel('Time (seconds)', fontsize=12)
ax1.set_title('MapReduce Performance: Serial vs Parallel\n(Per Phase Comparison)', fontsize=14)
ax1.set_xticks(x)
ax1.set_xticklabels(phases)
ax1.legend()
ax1.grid(axis='y', alpha=0.3)

# Add value labels on bars
for bar in bars1:
    height = bar.get_height()
    ax1.annotate(f'{height:.3f}s',
                xy=(bar.get_x() + bar.get_width() / 2, height),
                xytext=(0, 3),
                textcoords="offset points",
                ha='center', va='bottom', fontsize=10)

for bar in bars2:
    height = bar.get_height()
    ax1.annotate(f'{height:.3f}s',
                xy=(bar.get_x() + bar.get_width() / 2, height),
                xytext=(0, 3),
                textcoords="offset points",
                ha='center', va='bottom', fontsize=10)

# ============ Plot 2: Total time comparison with speedup ============
ax2 = axes[1]
categories = ['Serial\n(1 Task)', 'Parallel\n(5 Tasks)']
totals = [serial_total, parallel_total]
colors = ['#ff7f7f', '#7fbf7f']

bars = ax2.bar(categories, totals, color=colors, width=0.5)

ax2.set_ylabel('Total Time (seconds)', fontsize=12)
ax2.set_title('Total Execution Time Comparison', fontsize=14)
ax2.grid(axis='y', alpha=0.3)

# Add value labels
for bar, total in zip(bars, totals):
    height = bar.get_height()
    ax2.annotate(f'{total:.3f}s',
                xy=(bar.get_x() + bar.get_width() / 2, height),
                xytext=(0, 3),
                textcoords="offset points",
                ha='center', va='bottom', fontsize=12, fontweight='bold')

# Add speedup annotation
speedup = serial_total / parallel_total
ax2.annotate(f'Speedup: {speedup:.2f}x faster',
            xy=(0.5, max(totals) * 0.5),
            fontsize=14, fontweight='bold', color='green',
            ha='center')

# Map phase specific speedup
map_speedup = serial_data['Map (total)'] / parallel_data['Map (total)']
ax2.annotate(f'(Map phase: {map_speedup:.2f}x faster)',
            xy=(0.5, max(totals) * 0.35),
            fontsize=11, color='darkgreen',
            ha='center')

plt.tight_layout()
plt.savefig('mapreduce_performance.png', dpi=150, bbox_inches='tight')
plt.savefig('mapreduce_performance.pdf', bbox_inches='tight')
print("Charts saved as 'mapreduce_performance.png' and 'mapreduce_performance.pdf'")

# Also print summary
print("\n" + "="*50)
print("PERFORMANCE SUMMARY")
print("="*50)
print(f"Serial (1 Task):")
print(f"  Split:  {serial_data['Split']:.3f}s")
print(f"  Map:    {serial_data['Map (total)']:.3f}s (sequential)")
print(f"  Reduce: {serial_data['Reduce']:.3f}s")
print(f"  TOTAL:  {serial_total:.3f}s")
print()
print(f"Parallel (5 Tasks):")
print(f"  Split:  {parallel_data['Split']:.3f}s")
print(f"  Map:    {parallel_data['Map (total)']:.3f}s (3 mappers parallel)")
print(f"  Reduce: {parallel_data['Reduce']:.3f}s")
print(f"  TOTAL:  {parallel_total:.3f}s")
print()
print(f"SPEEDUP:")
print(f"  Map phase:   {map_speedup:.2f}x faster")
print(f"  Total time:  {speedup:.2f}x faster")
print("="*50)
