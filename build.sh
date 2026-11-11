#!/usr/bin/env bash
# Budowanie projektu na Ubuntu.
#
#   ./build.sh deps         instaluje pakiety (apt, wymaga sudo)
#   ./build.sh              = ./build.sh kod raport
#   ./build.sh kod          kompiluje program Go -> kod/itairodeh
#   ./build.sh bench [k]    pomiary (k seedów, domyślnie 20) -> raport/tabela*.tex
#   ./build.sh raport       raport/raport.pdf (latexmk + pdflatex)
#   ./build.sh prezentacja  prezentacja/prezentacja.html (osadza fonty Latin Modern)
#   ./build.sh demo         uruchamia przebieg z prezentacji (z ponowieniem)
#   ./build.sh all          kod bench raport prezentacja
#   ./build.sh clean        usuwa pliki pośrednie
set -euo pipefail
root="$(cd "$(dirname "$0")" && pwd)"

deps() {
  sudo apt-get update
  sudo apt-get install -y golang-go latexmk texlive-latex-recommended \
    texlive-latex-extra texlive-pictures texlive-lang-polish lmodern \
    fonts-lmodern python3
  check_go
}

check_go() {
  command -v go >/dev/null || { echo "brak Go — uruchom: ./build.sh deps" >&2; exit 1; }
  local v; v="$(go env GOVERSION | sed 's/^go//')"
  local major="${v%%.*}" minor; minor="$(echo "$v" | cut -d. -f2)"
  if [ "$major" -lt 1 ] || { [ "$major" -eq 1 ] && [ "$minor" -lt 21 ]; }; then
    echo "Go $v jest za stare (wymagane >= 1.21)." >&2
    echo "Na Ubuntu 22.04: sudo snap install go --classic" >&2
    exit 1
  fi
}

kod() {
  check_go
  (cd "$root/kod" && go vet . && go build -o itairodeh .)
  echo "ok: kod/itairodeh"
}

bench() {
  kod
  "$root/raport/bench.sh" "${1:-20}"
}

raport() {
  command -v latexmk >/dev/null || { echo "brak latexmk — uruchom: ./build.sh deps" >&2; exit 1; }
  [ -f "$root/raport/tabela.tex" ] && [ -f "$root/raport/tabela_c.tex" ] || bench
  (cd "$root/raport" && latexmk -pdf -interaction=nonstopmode -halt-on-error raport.tex >/dev/null)
  echo "ok: raport/raport.pdf"
}

prezentacja() {
  python3 "$root/prezentacja/build.py"
}

demo() {
  kod
  "$root/kod/itairodeh" -n 6 -N 4 -c 3 -wake one -seed 52 -delay "${1:-200ms}"
}

clean() {
  (cd "$root/raport" && latexmk -c raport.tex >/dev/null 2>&1 || true)
  rm -f "$root/kod/itairodeh"
}

case "${1:-}" in
  "")                 kod; raport ;;
  all)                kod; bench; raport; prezentacja ;;
  bench|demo)         t="$1"; shift; "$t" "$@" ;;
  deps|kod|raport|prezentacja|clean) "$1" ;;
  *) sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'; exit 1 ;;
esac
