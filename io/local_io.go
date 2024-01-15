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

func Local_fuzz_getchar() (r rune, err error) {
    Local_fuzz_flush()
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

func Local_fuzz_putchar(r rune) rune {
    len_buf := len(sbuf)
    cap_buf := cap(sbuf)
    if len_buf == cap_buf {
        Local_fuzz_flush()
        len_buf = 0
    }
    rune_len := 0
    if rune_len = utf8.RuneLen(r); rune_len == -1 {
        return 0
    }

    if room_avail := cap_buf - len_buf; room_avail < rune_len {
        Local_fuzz_flush()
    }
    utf8.AppendRune(sbuf, r)
    return r
}

func Local_fuzz_flush() {
    s := string(sbuf)
    OutputText(s)
    sbuf = s_data[:0]
}

func Local_fuzz_get_random() uint64 {
    var err error
    var randInt *big.Int
    max := big.NewInt(math.MaxInt64)
    if randInt, err = rand.Int(rand.Reader, max); err != nil {
        return 0
    }
    return randInt.Uint64()
}

func Local_fuzz_coin_flip() bool {
    n := Local_fuzz_get_random()
    return ((n % 2) == 0)
}


