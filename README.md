# python-image-pruner

Traverses a pip install and uses a greedy backoff algorithm + overlay fs + a venv to eliminate dead code. Uses a user-provided smoke test to determine whether the code is actually imported.

## Roadmap

- [x] Brute force testing modules one at a time
- [ ] Brute force testing multiple modules in parallel
- [ ] Using ptrace to see what files actually get opened by the python process
- [ ] Symbol-level pruning using tree-sitter to parse and replace files with only their top-level symbols

