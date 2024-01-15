package local

import (
  "bufio"
  "crypto/rand"
  "fmt"
  "golang.org/x/term"
  "math"
  "math/big"
  "os"
  "unicode/utf8"
)


// --------------------------------------------------------------------------------
// Local Handling for IO functions
// --------------------------------------------------------------------------------

// sbuf is a static buffer for getchar/putchar
var s_data [4096]byte
var sbuf []byte = s_data[:0]

func Fuzz_getchar() (r rune, err error) {
    Fuzz_flush()
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

func Fuzz_putchar(r rune) rune {
    len_buf := len(sbuf)
    cap_buf := cap(sbuf)
    if len_buf == cap_buf {
        Fuzz_flush()
        len_buf = 0
    }
    rune_len := 0
    if rune_len = utf8.RuneLen(r); rune_len == -1 {
        return 0
    }

    if room_avail := cap_buf - len_buf; room_avail < rune_len {
        Fuzz_flush()
    }
    utf8.AppendRune(sbuf, r)
    return r
}

func Fuzz_flush() {
    s := string(sbuf)
    // OutputText(s)
    Log_text(s, "info")
    sbuf = s_data[:0]
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


