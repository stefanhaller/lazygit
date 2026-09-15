package tasks

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Point LAZYGIT_REPLAY_DUMP at a file recorded by the stream dump (see
// stream_dump.go) to have this test print what that stream says and what
// lazygit makes of it. It reads the stream the way the task manager does and
// writes it to a view the way the task manager does, so a stream captured on
// one machine can be studied on another.
//
// The two halves of the output answer two different questions. The annotated
// stream says where the producer put each OSC 1717 record: a record belongs
// immediately before the text of the row it describes, with the row break
// before it and no row break between it and the text. The replayed view says
// which row each record ended up on, which is what every consumer of the
// metadata goes on.
const (
	replayDumpPathEnvVar   = "LAZYGIT_REPLAY_DUMP"
	replayDumpWidthEnvVar  = "LAZYGIT_REPLAY_WIDTH"
	replayDumpEventsEnvVar = "LAZYGIT_REPLAY_EVENTS"
)

func TestReplayStreamDump(t *testing.T) {
	path := os.Getenv(replayDumpPathEnvVar)
	if path == "" {
		t.Skipf("set %s to a recorded stream to replay it", replayDumpPathEnvVar)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	notes, data := parseStreamDump(t, contents)

	t.Logf("=== what was being read ===\n%s", strings.Join(notes, "\n"))

	width := replayWidth(t, notes)
	t.Logf("=== stream: %d bytes in %d reads, replayed at width %d ===",
		len(bytes.Join(data, nil)), len(data), width)

	t.Logf("=== annotated stream ===\n%s", strings.Join(annotateStream(bytes.Join(data, nil), width), "\n"))

	t.Logf("=== the view the stream produces ===\n%s", replayIntoView(data, width))
}

// parseStreamDump splits a recording into its notes and its raw data blocks,
// keeping the blocks separate so that the read boundaries stay visible.
func parseStreamDump(t *testing.T, contents []byte) (notes []string, data [][]byte) {
	t.Helper()

	for len(contents) > 0 {
		newline := bytes.IndexByte(contents, '\n')
		if newline == -1 {
			t.Fatalf("block header without a newline: %q", contents)
		}
		header := string(contents[:newline])
		contents = contents[newline+1:]

		switch {
		case strings.HasPrefix(header, streamDumpNotePrefix):
			notes = append(notes, strings.TrimPrefix(header, streamDumpNotePrefix))
		case strings.HasPrefix(header, streamDumpDataPrefix):
			count, err := strconv.Atoi(strings.TrimPrefix(header, streamDumpDataPrefix))
			if err != nil {
				t.Fatalf("unparseable data block length: %s", header)
			}
			if count+1 > len(contents) {
				t.Fatalf("truncated data block: %s promises more than the %d bytes left", header, len(contents))
			}
			data = append(data, contents[:count])
			// The recorder writes a newline after the bytes so that the next
			// header starts on its own line.
			contents = contents[count+1:]
		default:
			t.Fatalf("unrecognized block header: %q", header)
		}
	}

	return notes, data
}

// replayWidth is the content width to replay at: the one the recording states,
// unless LAZYGIT_REPLAY_WIDTH overrides it. The width decides where the view
// counts soft-wraps, so replaying at the wrong one invents row breaks that the
// machine the stream came from didn't have.
func replayWidth(t *testing.T, notes []string) int {
	t.Helper()

	if env := os.Getenv(replayDumpWidthEnvVar); env != "" {
		width, err := strconv.Atoi(env)
		if err != nil {
			t.Fatalf("%s must be a number: %s", replayDumpWidthEnvVar, env)
		}
		return width
	}

	for _, note := range notes {
		if _, rest, found := strings.Cut(note, "content width="); found {
			field, _, _ := strings.Cut(rest, ",")
			if width, err := strconv.Atoi(field); err == nil {
				return width
			}
		}
	}

	t.Fatalf("the recording doesn't state a content width; set %s", replayDumpWidthEnvVar)
	return 0
}

// replayIntoView feeds the stream through the task manager's reading pipeline
// into a view, and reports each row of the result with the records that landed
// on it. Rows are read back through DiffLineContents, which is what the
// consumers of the metadata use.
func replayIntoView(data [][]byte, width int) string {
	view := gocui.NewView("replay", 0, 0, width+1, 1000, gocui.OutputNormal)
	view.SetContentWidth(width)

	// The task manager scans the stream into lines and writes them to the view
	// one at a time, with the line's own terminator dropped and a newline
	// appended (see NewCmdTask).
	scanner := bufio.NewScanner(bytes.NewReader(bytes.Join(data, nil)))
	scanner.Split(utils.ScanLinesAndTruncateWhenLongerThanBuffer(bufio.MaxScanTokenSize))
	for scanner.Scan() {
		_, _ = view.Write(append(scanner.Bytes(), '\n'))
	}

	var result strings.Builder
	withRecords := 0
	contents := view.DiffLineContents()
	for i, content := range contents {
		if len(content.Metadata) > 0 {
			withRecords++
		}
		fmt.Fprintf(&result, "%4d  %-28s  %s\n", i, strings.Join(content.Metadata, " + "), content.Text)
	}
	fmt.Fprintf(&result, "\n%d of %d rows carry a record\n", withRecords, len(contents))
	return result.String()
}

// annotateStream renders a byte stream as one line per event, so that the
// placement of each escape sequence relative to the text and the row breaks
// can be read off. Text events carry the column the run ends at, because a run
// reaching the width is where a producer's row break and the view's soft-wrap
// have to agree.
func annotateStream(stream []byte, width int) []string {
	maxEvents := 400
	if env := os.Getenv(replayDumpEventsEnvVar); env != "" {
		if parsed, err := strconv.Atoi(env); err == nil {
			maxEvents = parsed
		}
	}

	var events []string
	row, col := 1, 1
	add := func(format string, args ...any) {
		events = append(events, fmt.Sprintf("r%-4d c%-4d  %s", row, col, fmt.Sprintf(format, args...)))
	}

	for i := 0; i < len(stream) && len(events) < maxEvents; {
		switch {
		case stream[i] == '\n':
			add("LF")
			row, col = row+1, 1
			i++
		case stream[i] == '\r':
			add("CR")
			col = 1
			i++
		case stream[i] == 0x1b && i+1 < len(stream) && stream[i+1] == ']':
			sequence, next := readOSC(stream, i)
			number, payload, _ := strings.Cut(sequence, ";")
			if number == "1717" {
				add("OSC 1717  %s", payload)
			} else {
				add("OSC %-5s %q", number, payload)
			}
			i = next
		case stream[i] == 0x1b && i+1 < len(stream) && stream[i+1] == '[':
			sequence, next := readCSI(stream, i)
			add("CSI  %-12s %s", sequence, csiName(sequence))
			i = next
		case stream[i] == 0x1b:
			add("ESC  %q", string(stream[i:min(i+2, len(stream))]))
			i += 2
		default:
			text, next := readText(stream, i)
			add("TEXT %q", text)
			col += len([]rune(text))
			if col > width+1 {
				events = append(events, fmt.Sprintf("%36s(past the width of %d)", "", width))
			}
			i = next
		}
	}

	if len(events) == maxEvents {
		events = append(events, fmt.Sprintf("(stopped after %d events; raise %s for more)",
			maxEvents, replayDumpEventsEnvVar))
	}
	return events
}

// readOSC returns the body of the OSC sequence starting at i (everything
// between the introducer and the terminator) and the index just past it. Both
// BEL and ESC \ terminate one.
func readOSC(stream []byte, i int) (string, int) {
	for j := i + 2; j < len(stream); j++ {
		switch {
		case stream[j] == 0x07:
			return string(stream[i+2 : j]), j + 1
		case stream[j] == 0x1b && j+1 < len(stream) && stream[j+1] == '\\':
			return string(stream[i+2 : j]), j + 2
		}
	}
	return string(stream[i+2:]) + " (unterminated)", len(stream)
}

// readCSI returns the CSI sequence starting at i without its introducer, and
// the index just past it. A byte in the final-byte range ends one.
func readCSI(stream []byte, i int) (string, int) {
	for j := i + 2; j < len(stream); j++ {
		if stream[j] >= 0x40 && stream[j] <= 0x7e {
			return string(stream[i+2 : j+1]), j + 1
		}
	}
	return string(stream[i+2:]) + " (unterminated)", len(stream)
}

// readText returns the run of ordinary characters starting at i and the index
// just past it.
func readText(stream []byte, i int) (string, int) {
	for j := i; j < len(stream); j++ {
		if stream[j] == 0x1b || stream[j] == '\n' || stream[j] == '\r' {
			return string(stream[i:j]), j
		}
	}
	return string(stream[i:]), len(stream)
}

// csiName names the sequences that move the cursor or erase, the ones that
// decide which row of the view the text after them lands on.
func csiName(sequence string) string {
	if sequence == "" {
		return ""
	}

	switch sequence[len(sequence)-1] {
	case 'H', 'f':
		return "cursor position"
	case 'A':
		return "cursor up"
	case 'B':
		return "cursor down"
	case 'C':
		return "cursor forward"
	case 'D':
		return "cursor back"
	case 'E':
		return "cursor next line"
	case 'F':
		return "cursor previous line"
	case 'G':
		return "cursor to column"
	case 'd':
		return "cursor to row"
	case 'J':
		return "erase in display"
	case 'K':
		return "erase in line"
	case 'X':
		return "erase characters"
	case 'L':
		return "insert lines"
	case 'M':
		return "delete lines"
	case 'm':
		return "" // colors and styles, which don't move the cursor
	default:
		return ""
	}
}
