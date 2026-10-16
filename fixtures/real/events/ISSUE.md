# Watch never gives up on an oversized event

`hrn remote watch` hangs on a job whose event stream carries one event line
over the 1 MiB limit. It reconnects every second and fetches the same oversized
event again, for as long as the watch lasts, and never reports what is wrong.

Expected: Watch stops at the first oversized event line and returns the error
ReadEvents gave for it, without a second request.
