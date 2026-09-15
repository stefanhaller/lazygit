# OSC 1717 diff metadata on Windows

What goes wrong with the diff metadata on Windows, what the evidence says, and
what is left to decide. Read `AGENTS.md` as well; everything in it applies here.

## The feature

A diff renderer that speaks the OSC 1717 protocol prefixes every row it renders
with a record saying which line of which file that row shows:

```
ESC ] 1717 ; version;type;new-line;old-line;file BEL
```

lazygit asks for these records by setting `OSC1717=V1` (`pkg/gui/pty.go`), reads
them out of the stream in `pkg/gocui/escape.go`, attaches them to the cells of
the view in `pkg/gocui/view.go`, and acts on them in
`pkg/gui/controllers/helpers/diff_line_parser.go`. They are what lets staging,
line selection, hunk navigation and "open in editor" work when a renderer has
restructured the diff so far that the rendering can no longer be parsed as one.

The protocol is specified at <https://github.com/stefanhaller/diff-line-metadata-spec>.
delta's emitter is <https://github.com/dandavison/delta/pull/2181>.

## The one invariant that matters

A record applies to the cells written after it, and stops applying at the end of
the line. So a record has to arrive **after the row break that ends the previous
row and before the text of the row it describes**. Nothing else about the stream
matters to the identity layer.

## The symptom

On Windows 11 build 26200, lazygit in Windows Terminal, patched delta: the first
hunk of a commit's diff is often not recognized at all, and the hunks that are
recognized are one line off. Neither happens on macOS with the same delta.

## What the evidence says

One commit's diff was captured on both platforms with `LAZYGIT_DUMP_STREAM` and
replayed through `TestReplayStreamDump`. The cause is settled.

**ConPTY delivers each OSC 1717 record as a write of its own, decoupled from the
text it belongs to.** conhost forwards a sequence it can't represent in its
screen buffer to the terminal side the moment it parses it, while the text goes
into the buffer and reaches the terminal side later, when a frame is painted. So
the records overtake the rows they describe.

The measurements, Windows against macOS:

| | macOS | Windows |
| --- | --- | --- |
| records in the stream | 148 | 148, byte-identical and in the same order |
| reads the stream arrived in | 22, median 1024 bytes | 24, median **32 bytes** |
| reads carrying records but no text | 0 | **13, holding 20 records** |
| rows of the view carrying a record | 147 of 180 | 128 of 180 |

Nothing is lost or garbled in transit. Only the position of each record relative
to the text differs, and that is the one thing the protocol depends on.

The two symptoms are two degrees of the same displacement:

- **The missing first hunk.** Twenty records — every record of the first file,
  plus the file header of the second — arrived before any of their text was
  painted, in thirteen back-to-back writes carrying no text at all. gocui keeps
  records that cover no cell by giving each one a zero-width carrier cell, so all
  twenty landed on row 1 of the view, and the rows they describe carry nothing.
  How many records get ahead of the painting depends on timing, which is why this
  comes and goes.
- **The rows one line off.** After the painting catches up, each record arrives
  just before the row break that ends the previous row rather than just after it.
  `finishLine` in `pkg/gocui/view.go` strands it on the row that is ending. The
  added file in the captured diff shows this on every row: macOS pairs the record
  `1;a;1` with the text `package tasks`, Windows pairs `1;a;2` with it.

A third effect is visible and would bite on its own. Windows encodes a blank row
as a cursor-position escape instead of a row break, and the `cursorDown` branch
of the write loop advances the row without calling `finishLine`, so a record
pending at that moment is discarded rather than stranded. Eleven blank rows in
this one diff came through that way.

**This cannot be repaired in gocui.** By the time lazygit reads the stream, the
adjacency that tied a record to its row is gone, and no rule about what to do
with a pending record can recover it.

(One difference between the two captures is not part of the cause. The macOS
delta ran with `--syntax-theme=none` and the Windows one did not. The record
sequences are identical, so it changes nothing here.)

## What is left to decide

The fix has to keep conhost out of the path between the renderer and lazygit.

1. **Run the renderer through a pipe on Windows rather than a pty.** The pty is
   there to make git invoke `GIT_PAGER`, so piping git's output into the renderer
   ourselves removes the need for it. lazygit already passes `--color=always` to
   git, so git's own colors don't depend on a terminal. What needs checking is
   whether each renderer still colorizes when its stdout is not a terminal. For
   delta on Windows, one command settles it:
   `git show --color=always <sha> | delta --paging=never > out.txt`, then look for
   SGR sequences and `1717` records in `out.txt`.
2. **Find out whether ConPTY can be told not to do this.** microsoft/terminal#1173
   asked for a passthrough mode and microsoft/terminal#17510 ("remove VtEngine",
   merged 2024-08-01) claims to have made passthrough the default. The captured
   stream is plainly not a passthrough of delta's bytes, so either a flag is
   missing on our `CreatePseudoConsole` call or that passthrough is narrower than
   it sounds. The evidence leans towards the decoupling being inherent to how
   conhost flushes a sequence it doesn't model, so treat this as a short
   experiment, not a plan. It would need a fallback for older Windows anyway.
3. Carrying the row number in the record would not help. conhost re-renders the
   rows themselves, so a row number from the renderer wouldn't describe the
   stream lazygit ends up reading.

Which of these to pursue is Stefan's call, made with the macOS session that has
the history of this work. Don't start on a fix here.

## Capturing and reading a stream

`LAZYGIT_DUMP_STREAM` records every byte lazygit reads from a command, framed so
the read boundaries survive, with a note per render giving the pty size and the
content width:

```
$env:LAZYGIT_DUMP_STREAM = "C:\tmp\windows.dump"
```

Every render appends, so keep a session to the one diff of interest and delete
the file between attempts. `TestReplayStreamDump` reads a dump back through the
same scanner and into a real view, on either platform:

```
LAZYGIT_REPLAY_DUMP=/path/to/windows.dump go test ./pkg/tasks/ -run TestReplayStreamDump -v
```

It prints the notes, the stream annotated one event per line with the row and
column each event happens at, and the resulting view with the records that landed
on each row. `LAZYGIT_REPLAY_EVENTS` raises the cap on annotated events (400 by
default) and `LAZYGIT_REPLAY_WIDTH` overrides the width the dump states.
Replaying at the wrong width invents soft-wraps the capturing machine didn't have.

## Ground rules on this machine

- The integration tests do not run on Windows at all. Don't try to use them, and
  don't spend the session fixing them.
- The unit tests do run on Windows, and CI runs them there, so `just unit-test`
  is the check that counts. `just build` builds. Use `just`, not `make`.
- This branch is a throwaway for the investigation. The feature stack it sits on
  is a series of pull requests under review; don't commit anything to those, and
  don't rewrite their history.
- The design notes for the whole feature are committed on the
  `fold-staging-functionality-into-main-view` branch as
  `focused-main-view-production-plan.md` and `diff-line-metadata-notes.md`. They
  are long. Search them; don't read them front to back.
