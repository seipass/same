# same

**Find out why it works on one machine and breaks on another.**

`same` captures two development environments and shows the differences most likely to matter.

```text
$ same compare works.json broken.json
Found 4 differences (3 high, 1 medium, 0 low)

! HIGH   git.commit
  working: 9d3e4c1
  broken:  7aa102e
  why:     The two machines are not running the same source revision.

! HIGH   tool.node.version
  working: v22.15.0
  broken:  v20.19.1
  why:     Different tool versions can change dependency resolution or runtime behavior.

! HIGH   env.DATABASE_URL.present
  working: true
  broken:  false
  why:     An environment variable exists on only one machine. Values are intentionally not stored.
```

## Use it

On the machine where the project works:

```sh
same capture -o works.json
```

On the machine where it breaks:

```sh
same capture -o broken.json
```

Then compare them:

```sh
same compare works.json broken.json
```

For machine-readable output:

```sh
same compare --json works.json broken.json
```

## What it compares

- operating system, CPU architecture, kernel, and timezone
- Git commit, branch, and uncommitted state
- installed developer tools, their versions, and resolved executable paths
- which environment variables exist
- a small allowlist of non-secret environment settings
- PATH order

`same` deliberately does **not** save environment variable values such as tokens, passwords, or connection strings. It only records whether those variables exist.

Home-directory paths are shortened to `~` in snapshots.

## Install

With Go:

```sh
go install github.com/seipass/same@latest
```

Or build from source:

```sh
git clone https://github.com/seipass/same
cd same
go build .
```

## Why

"Works on my machine" usually means the two machines are not actually the same. The hard part is finding the small difference that matters among hundreds that do not.

`same` starts with a deliberately simple rule: collect the common sources of environment drift, rank the differences, and keep secrets out of the snapshot.

## Status

Early release. Snapshot format is versioned, but may change before v1.0.

## License

MIT
