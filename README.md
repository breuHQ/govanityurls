# Go Vanity URLs

Go Vanity URLs is a fork of excellent work by [rakyll](https://github.com/rakyll). It is customized to work with `go.breu.io`.

## Getting started

After creating a new directory, edit `vanity.yaml` to add your repo. e.g., `go.breu.io/ctrlplane` simply add `/ctrlplane` and then the http path to github repo e.g.

```yaml
paths:
  /ctrlplane:
    repo: https://github.com/breuHQ/ctrlplane
    vcs: git
```

You can add as many rules as you wish.

## Configuration file

```yaml
host: example.com
cache_max_age: 3600 # in seconds
paths:
  /foo:
    repo: https://github.com/example/foo
    display: "https://github.com/example/foo https://github.com/example/foo/tree/master{/dir} https://github.com/example/foo/blob/master{/dir}/{file}#L{line}"
    vcs: git
```

| key           | required | default | description                                     |
| ------------- | -------- | ------- | ----------------------------------------------- |
| host          | yes      |         | the host e.g `example.com` or `go.breu.io` etc. |
| cache_max_age | no       | 86400   | default value for http cache-control header     |
| paths         | yes      |         | paths as described in path configuration below  |

### Path Configuration

| key     | required | description                                                                                                                                                                     |
| ------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| repo    | yes      | Root URL of the repository as it would appear in [go-import meta tag](https://golang.org/cmd/go/#hdr-Remote_import_paths).                                                       |
| vcs     | optional | can be `git`, `svn`, `bzr` & `hg`. if not provided, defaults to git.                                                                                                            |
| display | optional | The last three fields of the [go-source meta tag](https://github.com/golang/gddo/wiki/Source-Code-Links). If omitted, it is inferred from the code hosting service if possible. |
