package local

import (
  "bufio"
  "crypto/rand"
  "fmt"
  "golang.org/x/term"
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
    // [PH] Fuzz_flush()
    var state *term.State
    var stdin = int(os.Stdin.Fd())
    if state, err = term.MakeRaw(stdin); err != nil {
      return 0, err 
    }
    defer func() {
        if err = term.Restore(stdin, state); err != nil {
            fmt.Fprintln(os.Stderr, "warning, failed to restore terminal:", err)
        }
    }()

    in := bufio.NewReader(os.Stdin)
    r, _, err = in.ReadRune()
    return r, err
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
// [PH] 
// [PH] func Fuzz_flush() {
// [PH]     s := string(sbuf)
// [PH]     Log_text(s, "info")
// [PH]     sbuf = s_data[:0]
// [PH] }

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
