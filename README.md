# python-image-pruner

Traverses a python image and uses a greedy backoff algorithm + symlinks + a venv to eliminate dead code. Uses a smoke test to determine whether the code is actually imported. Reads the python backtrace to get an idea of the layer that is causing the error.
