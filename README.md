# gosh
An attempt at a POSIX compliant shell in Golang.
[![Build Status](https://drone.io/github.com/Danwakefield/gosh/status.png)](https://drone.io/github.com/Danwakefield/gosh/latest)

Currently only supports script files and lacks some important shell features like redirections, subshells and nested variables in some cases.

See the test-files folder for more examples of what currently works.

## License
Gosh is licensed under MIT.

# TODO
- [x] Pipeline Support - Requires changes to eval signature for passing IO redirections
- [ ] Redirections - Generic redirections to and from files, fd's, sockets etc.
- [ ] Background / Async commands - Should be quite easy just run Eval in goroutine and return ExitSuccess
- [ ] Functions - Requires variables.Scope to be updated to be a more generic store I.e not just variables but func / aliases
- [ ] Shebang - Preparse first line of a file.
- [ ] Subshells / backquotes - Subshell assignments have to be local which probably means changes to Scope.
- [ ] Fix naive parsing - Arith grabs upto its matching brackets but does not interpret any embedded arith or variables.
- [ ] Character escaping in strings
- [ ] Builtin commands - source / . will probably be first
- [ ] Interactive support - Use the golang readline port and add in prompts where needed
- [ ] Shell options - I.e set -x, prints line before evaluation. set -e exits on any non-zero status
- [x] Fix arithmetic ternary bug - See comments in file
