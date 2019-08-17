Package supervisor TODO.

There's `sync.WaitGroup`.  There's `golang.org/x/sync/errgroup.Group`.
There's `github.com/oklog/run.Group`.
`github.com/thejerf/suture.Supervisor`.  Why do we need another
goroutine-group library?

`sync.WaitGroup` is the smallest version.
`golang.org/x/sync/errgroup.Group` adds (1) error propagation and (2)
shutdown.  `github.com/oklog/run.Group` is capability-wise the same as
`golang.org/x/sync/errgroup.Group`, but doesn't use contexts (it does
things in a more pre-Go1.7 way).
`github.com/thejerf/suture.Supervisor` adds retry+backoff.  Features
you don't want but are there are bloat, and features you do want but
aren't there are limitations.

|                                              | error propagation | on goroutine return                                                  | goroutine notified of shutdown | retry+backoff    |
|----------------------------------------------|-------------------|----------------------------------------------------------------------|--------------------------------|------------------|
| sync.WaitGroup                               | no                | nothing                                                              | -                              |                  |
| golang.org/x/sync/errgroup                   | yes               | shutdown if non-nil err, nothing if nil                              | context                        |                  |
| github.com/oklog/run.Group                   | yes               | shutdown                                                             | callback                       |                  |
| github.com/thejerf/suture.Supervisor         | ?                 | retry (use Supervisor.Remove or implement IsCompletable to override) | callback                       | on normal return |
| github.com/datawire/teleproxy/pkg/supervisor | yes               | ?                                                                    | ?                              | ?                |

