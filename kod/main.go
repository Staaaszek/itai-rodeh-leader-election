// Itai & Rodeh (1990), §4.3.2: wybór lidera w asynchronicznym,
// jednokierunkowym, anonimowym pierścieniu, gdy znane jest tylko N
// takie, że N ≤ n ≤ 2N−1.
//
//	go run . -n 6 -N 4 -c 3 -wake one
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

type MsgType int

const (
	Wake MsgType = iota
	Bit
	Verify
	Result
	Count
	Size
)

type Role int

const (
	Sleep Role = iota
	Active
	Passive
	Candidate
	Leader
	Knows
)

type Message struct {
	Type  MsgType // Wake, Bit, Verify, Result, Count, Size
	Phase int     // numer fazy (ponowienia)
	Round int     // runda selekcji — do logów
	B     int     // Bit: wylosowany bit
	D, K  int     // Verify/Result: dystans, kandydaci
	U     bool    // Result: czy kandydat był jedyny
	J     int     // Count: licznik, Size: n
}

func (m Message) String() string {
	switch m.Type {
	case Bit:
		return fmt.Sprintf("Bit{ph:%d r:%d b:%d}", m.Phase, m.Round, m.B)
	case Verify:
		return fmt.Sprintf("Verify{ph:%d d:%d k:%d}", m.Phase, m.D, m.K)
	case Result:
		return fmt.Sprintf("Result{ph:%d d:%d u:%v}", m.Phase, m.D, m.U)
	case Count:
		return fmt.Sprintf("Count{j:%d}", m.J)
	case Size:
		return fmt.Sprintf("Size{j:%d}", m.J)
	}
	return "Wake"
}

// Stan jednego procesora — wszystko, co pamięta gorutyna
type NodeState struct {
	ID         int // tylko do logów — algorytm go NIE używa
	Phase      int // bieżąca faza
	Role       Role
	Round      int  // lokalna runda selekcji 1..c
	MyBit      int  // bit wylosowany w tej rundzie
	IsCand     bool // przetrwał c rund
	ResultSeen bool // zna już werdykt „jedyny”
	KnownN     int  // wynik b_v = n
	Halted     bool // terminacja procesora
	rng        *rand.Rand
}

type Config struct{ N, C int } // znane N, liczba rund c

// HandleMessage to cała logika węzła: dostaje jedną wiadomość,
// zmienia stan i zwraca wiadomości do następnego węzła.
func (s *NodeState) HandleMessage(m Message,
	cfg Config) (out []Message) {
	send := func(x Message) { out = append(out, x) }
	if s.Halted || (m.Type == Wake && s.Role != Sleep) {
		return // po zakończeniu / spóźniony wake-up
	}
	// budzę się / nowa faza
	if s.Role == Sleep || m.Phase > s.Phase {
		s.Phase, s.Role, s.Round = m.Phase, Active, 1
		s.IsCand, s.ResultSeen = false, false
		s.MyBit = s.rng.Intn(2)
		send(Message{Type: Bit, Phase: s.Phase,
			Round: 1, B: s.MyBit})
	}
	if m.Type == Wake || m.Phase < s.Phase {
		return // sam wake-up / wiadomość przeterminowana
	}
	switch m.Type {
	case Bit:
		if s.Role != Active {
			send(m) // pasywny tylko przekazuje
		} else if s.MyBit == 0 && m.B == 1 {
			s.Role = Passive // „0 po 1”: odpadam
		} else if s.Round < cfg.C {
			s.Round++ // przeżyłem rundę → następna
			s.MyBit = s.rng.Intn(2)
			send(Message{Type: Bit, Phase: s.Phase,
				Round: s.Round, B: s.MyBit})
		} else {
			s.Role, s.IsCand = Candidate, true // c rund
			send(Message{Type: Verify, Phase: s.Phase,
				D: 0, K: 1})
		}

	case Verify:
		d, k := m.D+1, m.K
		if s.IsCand {
			k = min(k+1, 3) // mijam kandydata
		}
		if d < 2*cfg.N-1 {
			send(Message{Type: Verify, Phase: s.Phase,
				D: d, K: k})
			return
		}
		// koniec trasy: ten węzeł zaczyna Result
		// i sam stosuje werdykt
		m = Message{Type: Result, Phase: s.Phase,
			D: -1, U: k == 2} // po D++ będzie 0
		fallthrough

	case Result:
		if m.D+1 < 2*cfg.N-1 { // też 2N−1 krawędzi
			m.D++
			send(m)
		}
		if !m.U { // nie jedyny: ponów w fazie p+1
			s.Phase, s.Role, s.Round = s.Phase+1, Active, 1
			s.IsCand, s.ResultSeen = false, false
			s.MyBit = s.rng.Intn(2)
			send(Message{Type: Bit, Phase: s.Phase,
				Round: 1, B: s.MyBit})
		} else if !s.ResultSeen { // kolejne kopie pomijam
			s.ResultSeen = true
			if s.IsCand {
				s.Role = Leader
				send(Message{Type: Count,
					Phase: s.Phase, J: 1})
			} else {
				s.Role = Knows
			}
		}

	case Count:
		if s.Role == Leader {
			s.KnownN = m.J // wróciło: n = J
			send(Message{Type: Size,
				Phase: s.Phase, J: m.J})
		} else {
			m.J++
			send(m)
		}

	case Size:
		if s.Role != Leader {
			s.KnownN = m.J
			send(m)
		}
		s.Halted = true // terminacja
	}
	return
}

func node(s *NodeState, cfg Config,
	in <-chan Message, out chan<- []Message) {
	for m := range in { // jedna gorutyna = jeden węzeł
		out <- s.HandleMessage(m, cfg)
		if s.Halted {
			return // procesor zakończył działanie
		}
	}
}

// Stats zlicza wiadomości i bity wysłane przez węzły (Wake pochodzi z zewnątrz).
type Stats struct {
	PerType [Size + 1]int
	Bits    int
	LogN    int // bity licznika do 2N
}

func (st *Stats) Count(m Message) {
	st.PerType[m.Type]++
	switch m.Type {
	case Bit:
		st.Bits++
	case Verify:
		st.Bits += st.LogN + 2
	case Result:
		st.Bits += st.LogN + 1
	default:
		st.Bits += st.LogN
	}
}

// Delay to globalne opóźnienie dostarczenia jednej wiadomości.
var Delay time.Duration

// pickNonEmpty losuje krawędź spośród tych z niepustym buforem.
func pickNonEmpty(queues [][]Message, rng *rand.Rand) (int, bool) {
	var ready []int
	for i, q := range queues {
		if len(q) > 0 {
			ready = append(ready, i)
		}
	}
	if len(ready) == 0 {
		return 0, false
	}
	return ready[rng.Intn(len(ready))], true
}

// wakeSet wybiera procesory, które dostają wake-up z zewnątrz.
func wakeSet(mode string, n int, rng *rand.Rand) []int {
	var w []int
	switch mode {
	case "all":
		for i := 0; i < n; i++ {
			w = append(w, i)
		}
	case "rand":
		for i := 0; i < n; i++ {
			if rng.Intn(3) == 0 {
				w = append(w, i)
			}
		}
	}
	if len(w) == 0 {
		w = []int{rng.Intn(n)}
	}
	return w
}

// run uruchamia n gorutyn i dyspozytora; po zakończeniu zwraca stany węzłów.
func run(n int, cfg Config, wake string, seed int64) ([]*NodeState, *Stats) {
	rng := rand.New(rand.NewSource(seed))
	stats := &Stats{LogN: int(math.Ceil(math.Log2(float64(2 * cfg.N))))}
	states := make([]*NodeState, n)
	inbox := make([]chan Message, n)
	outbox := make([]chan []Message, n)
	queues := make([][]Message, n) // queues[j]: krawędź v(j−1) → vj
	var wg sync.WaitGroup
	for i := range states {
		states[i] = &NodeState{ID: i, rng: rand.New(rand.NewSource(seed + int64(i) + 1))}
		inbox[i] = make(chan Message)
		outbox[i] = make(chan []Message)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			node(states[i], cfg, inbox[i], outbox[i])
		}(i)
	}
	for _, i := range wakeSet(wake, n, rng) {
		log.Printf("wake-up v%d", i)
		queues[i] = append(queues[i], Message{Type: Wake})
	}

	// main: bufory FIFO krawędzi + losowy harmonogram
	for {
		i, ok := pickNonEmpty(queues, rng) // losowa krawędź
		if !ok {
			break // wszystkie bufory puste → koniec
		}
		m := queues[i][0]
		queues[i] = queues[i][1:]
		inbox[i] <- m // kanał bez bufora
		for _, x := range <-outbox[i] {
			j := (i + 1) % n
			queues[j] = append(queues[j], x)
			log.Printf("v%d wysłał %v do v%d", i, x, j)
			stats.Count(x)
		}
		time.Sleep(Delay) // globalne opóźnienie
	}

	for i := range inbox {
		close(inbox[i])
	}
	wg.Wait()
	return states, stats
}

// summary wypisuje podsumowanie i zwraca, czy wynik jest poprawny.
func summary(states []*NodeState, stats *Stats, cfg Config) bool {
	n := len(states)
	leader, leaders, known, halted := -1, 0, 0, 0
	for i, s := range states {
		if s.Role == Leader {
			leader, leaders = i, leaders+1
		}
		if s.KnownN == n {
			known++
		}
		if s.Halted {
			halted++
		}
	}
	t, total := stats.PerType, 0
	for _, c := range t {
		total += c
	}
	phases := states[0].Phase + 1
	fmt.Println("=== PODSUMOWANIE ===")
	fmt.Printf("n=%d N=%d c=%d\n", n, cfg.N, cfg.C)
	fmt.Printf("wiadomości: %d (Bit %d, Verify %d, Result %d, Count %d, Size %d)\n",
		total, t[Bit], t[Verify], t[Result], t[Count], t[Size])
	fmt.Printf("bity: %d, fazy: %d\n", stats.Bits, phases)
	fmt.Printf("lider: v%d, liderów: %d, znają n: %d/%d, zakończyło: %d/%d\n",
		leader, leaders, known, n, halted, n)
	return leaders == 1 && known == n && halted == n
}

func main() {
	nodes := flag.Int("n", 6, "rzeczywisty rozmiar pierścienia")
	bigN := flag.Int("N", 4, "znane N, musi być N ≤ n ≤ 2N−1")
	rounds := flag.Int("c", 0, "liczba rund selekcji; 0 = ⌈2 log_{4/3}(2N)⌉")
	wake := flag.String("wake", "one", "kto startuje: one, all, rand")
	seed := flag.Int64("seed", time.Now().UnixNano(), "ziarno losowości")
	flag.DurationVar(&Delay, "delay", 10*time.Millisecond, "globalne opóźnienie")
	quiet := flag.Bool("quiet", false, "bez logu wiadomości, tylko podsumowanie")
	flag.Parse()

	log.SetFlags(0)
	log.SetOutput(os.Stdout)
	n, cfg := *nodes, Config{N: *bigN, C: *rounds}
	if cfg.N < 1 || n < cfg.N || n > 2*cfg.N-1 {
		log.Fatalf("wymagane N ≤ n ≤ 2N−1, a jest n=%d, N=%d", n, cfg.N)
	}
	if cfg.C == 0 {
		cfg.C = int(math.Ceil(2 * math.Log(float64(2*cfg.N)) / math.Log(4.0/3)))
	}
	// w jednej rundzie odpada najwyżej połowa aktywnych
	if math.Pow(2, float64(cfg.C)) < float64(n) {
		log.Fatalf("c=%d: przy 2^c < n żadna faza się nie uda", cfg.C)
	}
	log.Printf("seed=%d", *seed)
	if *quiet {
		log.SetOutput(io.Discard)
	}

	states, stats := run(n, cfg, *wake, *seed)
	if !summary(states, stats, cfg) {
		os.Exit(1)
	}
}
