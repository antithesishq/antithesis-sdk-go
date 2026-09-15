//go:build no_antithesis_sdk

package assert

// Enabled reports whether the Antithesis SDK was compiled into this program.
// See the documentation on the variant of this file built without the
// no_antithesis_sdk build tag.
const Enabled = false
