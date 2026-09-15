//go:build !enable_antithesis_sdk

package assert

// Enabled reports whether the Antithesis SDK was compiled into this program.
// See the documentation on the variant of this file built with the
// enable_antithesis_sdk build tag.
const Enabled = false
