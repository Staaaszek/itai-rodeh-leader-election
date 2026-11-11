# Itai & Rodeh (1990), §4.3.2 — asynchroniczny wybór lidera przy przybliżonym n

Wybór lidera w asynchronicznym, jednokierunkowym, anonimowym pierścieniu, gdy znane jest tylko `N` takie, że `N ≤ n ≤ 2N−1`
(A. Itai, M. Rodeh, *Symmetry Breaking in Distributed Networks*).

## Co gdzie jest

| Ścieżka | Zawartość |
|---|---|
| `prezentacja/prezentacja.html` | prezentacja — otworzyć w przeglądarce (strzałki / spacja, `H` — pomoc) |
| `prezentacja/prezentacja.src.html`, `prezentacja/build.py` | źródło slajdów i skrypt osadzający fonty |
| `kod/main.go` | implementacja w Go: `NodeState` (stan węzła), `HandleMessage` (obsługa jednej wiadomości), `node` (gorutyna), `run` (dyspozytor z kolejkami FIFO i losowym harmonogramem) |
| `raport/raport.pdf` | raport |
| `raport/raport.tex`, `raport/tabela*.tex` | źródło raportu i wygenerowane tabele pomiarów |
| `raport/bench.sh` | pomiary skalowania (generuje `raport/tabela*.tex`) |
| `build.sh` | budowanie i uruchamianie na Ubuntu |

## Jak uruchomić

Wymagany Go ≥ 1.21.

```bash
./build.sh deps    # jednorazowo: instaluje Go (apt, sudo); na Ubuntu 22.04: sudo snap install go --classic
./build.sh         # kompilacja -> kod/itairodeh
./build.sh demo    # przebieg z prezentacji: n=6, N=4, c=3, jedno ponowienie
```

Albo bezpośrednio:

```bash
cd kod
go run . -n 6 -N 4 -c 3 -wake one -seed 52 -delay 200ms   # demo z ponowieniem
go run . -n 1024 -N 600 -wake rand -quiet               # tylko podsumowanie
```

Flagi:

| Flaga | Znaczenie |
|---|---|
| `-n` | rzeczywisty rozmiar pierścienia |
| `-N` | znane `N`, wymagane `N ≤ n ≤ 2N−1` |
| `-c` | liczba rund selekcji; `0` = domyślne `⌈2 log_{4/3}(2N)⌉` (odrzucane, gdy `2^c < n`) |
| `-wake` | kto startuje: `one`, `all`, `rand` |
| `-seed` | ziarno (harmonogram i rzuty monetą — przebieg jest powtarzalny) |
| `-delay` | globalne opóźnienie między dostarczeniami wiadomości (domyślnie `10ms`) |
| `-quiet` | bez logu wiadomości, tylko podsumowanie |

Log pokazuje każdą wiadomość (`vX wysłał M do vY`), na końcu podsumowanie: liczba wiadomości wg typu, bity, fazy,
lider i czy wszyscy znają `n`. Przy niepoprawnym wyniku program kończy się kodem 1.

Pozostałe cele `build.sh` (wymagają LaTeX-a / Pythona, gotowe pliki są już w repo):
`./build.sh bench` — ponowne pomiary, `./build.sh raport` — `raport/raport.pdf`, `./build.sh prezentacja` — `prezentacja/prezentacja.html`.
