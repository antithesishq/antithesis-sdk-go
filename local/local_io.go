package local

import (
  "crypto/rand"
  "math"
  "math/big"
  "os"
  // [PH] "unicode/utf8"
)


// --------------------------------------------------------------------------------
// Local Handling for IO functions
// --------------------------------------------------------------------------------

// sbuf is a static buffer for getchar/putchar
// [PH] var s_data [4096]byte
// [PH] var sbuf []byte = s_data[:0]

func Fuzz_getchar() (r rune, err error) {
    var num uint64 = Fuzz_get_random()
    var one_byte rune = rune(num & 0x000000ff)
    return one_byte, nil
    // [PH] Fuzz_flush()
    // [PH] var state *term.State
    // [PH] var stdin = int(os.Stdin.Fd())
    // [PH] if state, err = term.MakeRaw(stdin); err != nil {
    // [PH]   return 0, err 
    // [PH] }
    // [PH] defer func() {
    // [PH]     if err = term.Restore(stdin, state); err != nil {
    // [PH]         fmt.Fprintln(os.Stderr, "warning, failed to restore terminal:", err)
    // [PH]     }
    // [PH] }()
    // [PH] in := bufio.NewReader(os.Stdin)
    // [PH] r, _, err = in.ReadRune()
    // [PH] return r, err
}

// [PH] func Fuzz_putchar(r rune) rune {
// [PH]     len_buf := len(sbuf)
// [PH]     cap_buf := cap(sbuf)
// [PH]     if len_buf == cap_buf {
// [PH]         Fuzz_flush()
// [PH]         len_buf = 0
// [PH]     }
// [PH]     rune_len := 0
// [PH]     if rune_len = utf8.RuneLen(r); rune_len == -1 {
// [PH]         return 0
// [PH]     }
// [PH] 
// [PH]     if room_avail := cap_buf - len_buf; room_avail < rune_len {
// [PH]         Fuzz_flush()
// [PH]     }
// [PH]     utf8.AppendRune(sbuf, r)
// [PH]     return r
// [PH] }

func Fuzz_flush() {
    // [PH] s := string(sbuf)
    // [PH] Log_text(s, "info")
    // [PH] sbuf = s_data[:0]
}

func Fuzz_get_random() uint64 {
    var err error
    var randInt *big.Int
    max := big.NewInt(math.MaxInt64)
    if randInt, err = rand.Int(rand.Reader, max); err != nil {
        return 0
    }
    return randInt.Uint64()
}

func Fuzz_coin_flip() bool {
    n := Fuzz_get_random()
    return ((n % 2) == 0)
}
