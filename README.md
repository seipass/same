# same

**"Works on my machine." Find out why.**

`same` compares a working machine with a broken one and shows the differences most likely to matter.

```text
working machine          broken machine
      │                         │
      └── same capture          └── same capture
                 \             /
                  same compare
                       │
                       ▼
              the useful differences
```

## Try it in 30 seconds

On the machine where the project works:

```sh
same capture -o works.json
```

On the machine where it breaks:

```sh
same capture -o broken.json
```

Compare them:

```sh
same compare works.json broken.json
```

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
  why:     An environment variable exists on only one machine.
```

That's the whole idea: **capture both machines, compare them, fix the difference that matters.**

## Install

```sh
go install github.com/seipass/same@latest
```

Or build it locally:

```sh
git clone https://github.com/seipass/same
cd same
go build .
```

## What `same` compares

| Area | Examples |
| --- | --- |
| System | operating system, CPU architecture, kernel, timezone |
| Source | Git commit, branch, uncommitted changes |
| Tools | Node, Python, Go, Java, package managers, executable paths |
| Environment | which environment variables exist |
| PATH | search order and resolved tool locations |

The output is ranked so the most suspicious differences appear first.

## Secrets stay out

`same` **does not store environment variable values**.

It records whether a variable exists, so this:

```text
DATABASE_URL: present
```

can be compared without putting the connection string into a snapshot.

Home-directory paths are shortened to `~` as well.

## Machine-readable output

```sh
same compare --json works.json broken.json
```

This makes `same` usable in scripts, bug reports, and automated checks.

## Why this exists

When software works on one machine and fails on another, the machines differ somewhere.

The annoying part is finding the one useful difference among hundreds of irrelevant ones.

`same` collects the common sources of environment drift and puts the likely causes at the top.

## Status

Early release. The snapshot format is versioned and may change before v1.0.

## License

MIT
