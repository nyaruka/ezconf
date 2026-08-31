v0.7.0 (2026-08-31)
-------------------------
 * Fix silently dropped config values when the destination isn't a settable pointer to a struct
 * Return errors from Load instead of exiting the process, with -help and -h returning flag.ErrHelp and Usage exported
 * Reject reserved field names with an error instead of panicking inside the flag package
 * Parse float64 fields at full precision instead of truncating to float32
 * Remove the undocumented -debug-conf flag which printed resolved config values including credentials
 * Require Go 1.25

v0.6.1 (2026-02-17)
-------------------------
 * Use real CSV parsing

v0.6.0 (2026-02-17)
-------------------------
 * Add support for arrays of strings and ints encoded as CSV

v0.5.0 (2026-02-16)
-------------------------
 * Add support for name struct field tag to control snakification

v0.4.1 (2025-09-08)
-------------------------
 * Don't fail build for codecov failures because it always fails argh

v0.4.0 (2025-09-05)
-------------------------
 * Rename EZLoader to just Loader
 * Allow overriding of args via SetArgs
 * Update to go 1.24 and update deps

v0.3.0 (2024-01-16)
-------------------------
 * Require go 1.21
 * Add support for slog.Level fields

