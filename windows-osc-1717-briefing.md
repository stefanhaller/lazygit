# OSC 1717 diff metadata on Windows

A briefing for a session picking this up on the Windows machine. It states the
problem, what is already established about it, and what to do next. Read
`AGENTS.md` as well; everything in it applies here.

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
Without them lazygit falls back to parsing the rendered text as a unified diff,
which under delta fails, so the rows resolve to nothing.

The protocol is specified at <https://github.com/stefanhaller/diff-line-metadata-spec>.
delta's emitter is <https://github.com/dandavison/delta/pull/2181>.

## The one invariant that matters

A record applies to the cells written after it, and stops applying at the end of
the line. So a record has to arrive **after the row break that ends the previous
row and before the text of the row it describes**. Nothing else about the stream
matters to the identity layer.

On Unix lazygit reads the renderer's own bytes, so the invariant holds by
construction. On Windows lazygit reads what ConPTY makes of those bytes, and
whether the invariant survives that is the open question.

## The symptom

Reported on Windows 11 build 26200, lazygit running in Windows Terminal, with
the patched delta. Two failures, both intermittent:

1. Hunks are not recognized at all.
2. When they are recognized, they are often one line off.

Neither happens on macOS with the same delta and the same diffs.

## What is established

**A displacement of the record across the row break produces exactly these two
symptoms.** This was reproduced on macOS by replaying synthetic streams through
the real reading pipeline (see "How to read a dump" below):

| What the stream does with the record | What lazygit does with it |
| --- | --- |
| record, then the row's text, then LF | correct; every row carries its record |
| row's text, then the **next** row's record, then LF | every row reports the identity of the row below it, and the last row reports none — **symptom 2** |
| record, then a cursor-position escape, then the text | the record is destroyed and the row carries none — **symptom 1** |
| a cursor-position escape, then the record, then the text | correct |
| record, then CR, then text, then LF | correct |
| CRLF row breaks instead of LF | correct |

The two failing shapes differ from the working ones only in that the record
arrives before the row break rather than after it. The reason the two failures
look so different is the kind of row break involved. When the break is a
newline, `finishLine` in `pkg/gocui/view.go` keeps the stranded record by giving
it a cell on the row that is ending, so it survives on the wrong row. When the
break is a cursor-position escape, the `cursorDown` branch of the write loop
advances without calling `finishLine`, and `advanceToNextLine` resets the
pending record, so it is lost.

**ConPTY is not obviously the culprit any more, but it is still the prime
suspect.** microsoft/terminal#1173 asked for a passthrough mode; it is closed,
and microsoft/terminal#17510 ("remove VtEngine") merged on 2024-08-01 made
ConPTY forward an application's VT output unmodified rather than re-rendering it
through a screen buffer. Build 26200 is well past that, so delta's records ought
to arrive verbatim. Two things keep the suspicion alive. That PR's own notes call
out a workaround for delayed end-of-line wrapping as having broken reflow, and a
deferred row break is precisely a row break that can end up on the wrong side of
a record. And `pkg/gocui/escape.go` already carries machinery for ConPTY
emitting cursor-position escapes in place of newlines to skip blank rows, so
that shape does occur.

This yields a prediction worth checking first: the displacement sets in on the
row after one whose text reaches the full width of the pty, because that is the
row whose break ConPTY has to synthesize rather than forward.

## What to do first: capture the stream

`LAZYGIT_DUMP_STREAM` records every byte lazygit reads from a command, framed so
that the read boundaries survive, with a note per render giving the pty size and
the content width. Capture a bad case on Windows:

```
$env:LAZYGIT_DUMP_STREAM = "C:\tmp\windows.dump"
.\lazygit.exe
```

Select a commit whose diff misbehaves, confirm it misbehaves, and quit. Every
main-view render appends to the file, so keep the session to the one diff, and
delete the file between attempts. Note down what you saw: whether the rows
resolved to nothing or to the row below, and at which row it started.

Then capture the same diff of the same repository on macOS, where it works. Two
dumps of one diff from the two platforms answer the question by subtraction: the
difference between the streams is the bug, and everything else can be ruled out.

## How to read a dump

`TestReplayStreamDump` in `pkg/tasks/stream_dump_replay_test.go` reads a dump
back through the same scanner and into a real view, on either platform:

```
LAZYGIT_REPLAY_DUMP=/path/to/windows.dump go test ./pkg/tasks/ -run TestReplayStreamDump -v
```

It prints three things. The notes say what was being read and at what width. The
annotated stream gives one line per event, with the row and column each event
happens at, so the placement of every record relative to the text and the row
breaks can be read off directly; a text run that reaches the width is flagged.
The replayed view lists each row of the result with the records that landed on
it, which is what every consumer of the metadata sees. A row whose text says
line 42 while its record says 41 is symptom 2; a row with no record at all is
symptom 1.

`LAZYGIT_REPLAY_EVENTS` raises the cap on annotated events (400 by default), and
`LAZYGIT_REPLAY_WIDTH` overrides the width the dump states. Replaying at the
wrong width invents soft-wraps the capturing machine didn't have.

## Where the fix will go, and who decides

If the dump shows the record displaced, the question is whether to stop ConPTY
producing the displacement or to make the record-to-row binding in
`pkg/gocui/view.go` tolerate it. The second touches code that Unix shares, and
the first may mean not running diff renderers in a pty on Windows at all, which
is a change to how every renderer is invoked. Both are calls for Stefan to make
with the macOS session that has the full history of this work, per the
"Surface mid-implementation decisions" section of `AGENTS.md`.

So: capture the dumps, run the replay, report what the two halves of its output
say, and stop there. Do not design or apply a fix, and do not change the
protocol or the emitter.

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
