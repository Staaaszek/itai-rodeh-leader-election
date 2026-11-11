#!/usr/bin/env bash
# Generuje tabelę skalowania (raport/tabela.tex) z programu w kod/.
# Użycie: raport/bench.sh [liczba_seedów]   (domyślnie 20)
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
bin="$here/../kod/itairodeh"
seeds="${1:-20}"
[ -x "$bin" ] || (cd "$here/../kod" && go build -o itairodeh .)

out="$here/tabela.tex"
: > "$out"
for mode in tight loose; do
  for n in 8 16 32 64 128 256 512 1024; do
    if [ "$mode" = tight ]; then N=$n; else N=$(( (n + 2) / 2 )); fi
    for s in $(seq 1 "$seeds"); do
      "$bin" -n "$n" -N "$N" -wake rand -delay 0 -quiet -seed "$s"
    done | awk -v n="$n" -v N="$N" -v k="$seeds" '
      /^n=/          { split($3, a, "="); c = a[2] }
      /^wiadomości:/ { gsub(/[(),]/, ""); tot += $2; bit += $4; ver += $6; res += $8; cnt += $10 + $12 }
      /^bity:/       { gsub(/,/, ""); bits += $2; ph += $4 }
      /^lider:/      { gsub(/,/, ""); if ($4 == "1" && $7 == n"/"n && $9 == n"/"n) ok++ }
      END {
        l = n * log(n) / log(2)
        printf "%d & %d & %d & %.0f & %.0f & %.0f & %.0f & %.0f & %.0f & %.2f & %.2f & %.2f & %d/%d \\\\\n",
          n, N, c, bit/k, ver/k, res/k, cnt/k, tot/k, bits/k, ph/k, tot/k/l, bits/k/l, ok, k
      }' >> "$out"
  done
  [ "$mode" = tight ] && echo '\midrule' >> "$out"
done

# wpływ c przy stałym n = N = 64: mniej rund selekcji ⇒ więcej ponowień
out2="$here/tabela_c.tex"
: > "$out2"
for c in 10 11 12 14 17 20 34; do
  for s in $(seq 1 "$seeds"); do
    "$bin" -n 64 -N 64 -c "$c" -wake rand -delay 0 -quiet -seed "$s"
  done | awk -v c="$c" -v k="$seeds" '
    /^wiadomości:/ { gsub(/[(),]/, ""); tot += $2; bit += $4; ver += $6 + $8 }
    /^bity:/       { gsub(/,/, ""); bits += $2; ph += $4; if ($4 > mx) mx = $4 }
    END { printf "%d & %.2f & %d & %.0f & %.0f & %.0f & %.0f \\\\\n", c, ph/k, mx, bit/k, ver/k, tot/k, bits/k }' >> "$out2"
done
cat "$out" "$out2"
