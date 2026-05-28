# Sample workspace

This is the workspace that boots inside your ephemeral container.

- `public/index.html` is served at the `/preview` route — edit it and the preview reloads.
- Open the terminal and try `ls`, `git --version`, `echo $WORKSPACE_DIR`.

When the container sleeps the workspace resets. Persistence is a v2 problem.
