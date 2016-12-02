# gosh
An attempt at a POSIX compliant shell in Golang.

Currently only supports script files and lacks some important shell features like redirections, subshells and nested variables in some cases.
See the test-files folder for more examples of what currently works.

Not perfect but it is currently in code freeze for submission.
I think the biggest problem right now are the circular dependencies.
Move Scope from the variables package into main and figure some way for arith to continue working


Uses [Govend](https://github.com/govend/govend) for vendoring.
This will only matter if you add a dependency and if you would like
you can manually copy the code and edit vendor.yml to contain the revision ID

## License
Gosh is licensed under MIT.

# TODO
- [ ] Word splitting
- [ ] filepath globbing
- [ ] Redirections - Generic redirections to and from files, fd's, sockets etc.
- [ ] Background / Async commands - Should be quite easy just run Eval in goroutine and return ExitSuccess
- [ ] backquotes
- [ ] Fix naive parsing - Arith grabs upto its matching brackets but does not interpret any embedded arith or variables.
- [ ] Character escaping in strings
- [ ] Interactive support - Use the golang readline port and add in prompts where needed
- [ ] Shell options - I.e set -x, prints line before evaluation. set -e exits on any non-zero status
- [x] Switch to a log library (write one?) that follows [Dave Cheneys blog post](http://dave.cheney.net/2015/11/05/lets-talk-about-logging) ideas. See https://github.com/danwakefield/kisslog
- [x] Shebang - Preparse first line of a file. (Done by exec.Command)
- [x] tilde expansion
- [x] Builtin commands - source / . will probably be first
- [x] Functions - Requires variables.Scope to be updated to be a more generic store I.e not just variables but func / aliases
- [x] Pipeline Support - Requires changes to eval signature for passing IO redirections
- [x] Fix arithmetic ternary bug - See comments in file
- [x] Subshells
