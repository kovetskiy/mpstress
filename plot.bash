#!/bin/bash

set -euo pipefail

common="
set xdata time
set timefmt \"%H:%M:%S\"
set format x \"%H:%M:%S\"

set ytics 10
set grid
set style data lines
set term png size 1200, 500
set yrange [0:100]
"

for host in $(find ./output/ -maxdepth 1 -type d -printf '%P\n' | sort -n); do
    gnuplot <<HEAD
set title "$host"

${common}

set output "output/$host.png"
plot \
    "output/$host/total" using 1:2 title "total" with lines, \\
    "output/$host/usr" using 1:2 title "usr" with lines, \\
    "output/$host/sys" using 1:2 title "sys" with lines
HEAD

done

summary="${common}
set output \"output/summary_total.png\"

plot \\"
for host in $(find ./output/ -maxdepth 1 -type d -printf '%P\n' | sort -n); do
    summary="$summary
    \"output/$host/total\" using 1:2 title \"$host cpu\" with lines, \\"
done

gnuplot <<< "$summary"
