# same

**Your project works on one machine and fails on another. `same` shows you the environment differences worth checking first.**

`same` takes a small snapshot of each development environment, then compares the two snapshots.

It looks at things such as:

- the Git commit and branch
- Node, Python, Go, Java, Rust, package-manager and other tool versions
- operating system and CPU architecture
- whether important environment variables exist
- PATH order
- timezone and a few other machine settings

It does **not** store environment-variable values such as passwords, tokens, or connection strings.

```text
Machine A: project works            Machine B: project fails

same capture -o works.json          same capture -o broken.json
             │                                   │
             └──────── copy both JSON files ─────┘
                              │
                              ▼
              same compare works.json broken.json
                              │
                              ▼
                  differences ranked by importance
```

Example:

```text
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
  why:     The variable exists on only one machine. Its value was not recorded.
```

`same` does not claim that every difference it shows is the cause. It puts the differences most likely to matter at the top so you have a short list to investigate.

## Install

```sh
go install github.com/seipass/same@latest
```

Or build it from source:

```sh
git clone https://github.com/seipass/same
cd same
go build .
```

## Quick start

On the machine where the project works:

```sh
same capture -o works.json
```

On the machine where it fails:

```sh
same capture -o broken.json
```

Copy the two JSON files onto either machine, then compare them:

```sh
same compare works.json broken.json
```

For machine-readable output:

```sh
same compare --json works.json broken.json
```

## What a snapshot contains

A snapshot records:

- operating system, CPU architecture, kernel, and timezone
- Git commit, branch, and whether the working tree has local changes
- detected developer tools, their versions, and the executable paths being used
- which environment-variable names are present
- a small set of non-secret environment settings
- PATH order

Home-directory paths are shortened to `~`.

Environment-variable values are intentionally omitted.

## What `same` is for

Use `same` when the same project behaves differently across two machines and you suspect the environments have drifted apart.

Typical examples:

- one developer can run the project and another cannot
- a local machine behaves differently from a CI runner
- an old laptop works while a newly set-up laptop does not
- two machines appear identical but resolve different tools or versions

`same` narrows the search to concrete differences you can inspect.

## Status

Early release. The snapshot format is versioned and may change before v1.0.

## License

MIT
