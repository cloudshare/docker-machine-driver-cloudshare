# Building

- Install Go
- Clone this repo
- `make build` (will build for your local arch/os)


# Release (upload binaries to GitHub)

- Commit, tag and push (i.e. push the tag as well: `git push --tags`...)
- Make sure you have `github-release` installed:
    - `go get github.com/c4milo/github-release`
    - `GITHUB_TOKEN` should be defined with your personal access token.
        - You need to have push permissions to this repo.
        - The token should have `repo` scope. See: https://i.imgur.com/3cdkgG2.png
- `make package` will build for all supported OSs and create tar.gz files in `dist/`.
- To upload the tag (e.g. if the tag is `3.2.1`) do:
   -  `TAG=3.2.1 make upload`

