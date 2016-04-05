# gosh
Go Shell. Final year project at Aberystwyth University.
The aim of GOSH is to create a POSIX compliant shell runtime using a modern type safe language


# TODO
- [x] Pipeline Support - Requires changes to eval signature for passing IO redirections
- [ ] Redirections - Generic redirections to and from files, fd's, sockets etc.
- [ ] Background / Async commands - Should be quite easy just run Eval in goroutine and return ExitSuccess
- [ ] Functions - Requires variables.Scope to be updated to be a more generic store I.e not just variables but func / aliases
- [ ] Shebang - Preparse first line of a file.
- [ ] Subshell / Backquotes - Naive ones could be done by eating upto the next \`/). Complex by a recursive Parser
- [ ] Character escaping in strings
- [ ] Builtin commands - source / . will probably be first
- [ ] Interactive support - Use the golang readline port and add in prompts where needed
- [ ] Shell options - I.e set -x, prints line before evaluation. set -e exits on any non-zero status
- [ ] Fix arithmetic ternary bug - See comments in file
