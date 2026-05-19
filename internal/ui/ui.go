package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/muesli/termenv"
)

type Options struct {
	Plain bool
	Color string
}

type UI struct {
	out   io.Writer
	plain bool

	heading lipgloss.Style
	success lipgloss.Style
	warning lipgloss.Style
	failure lipgloss.Style
	muted   lipgloss.Style
	label   lipgloss.Style
	value   lipgloss.Style
	profile lipgloss.Style
	path    lipgloss.Style
	command lipgloss.Style
}

type CheckState string

const (
	CheckOK   CheckState = "OK"
	CheckWarn CheckState = "WARN"
	CheckFail CheckState = "FAIL"
)

func New(out io.Writer, opts Options) *UI {
	forceColor := opts.Color == "always" || os.Getenv("FORCE_COLOR") != ""
	plain := opts.Plain || opts.Color == "never" || (!forceColor && (os.Getenv("NO_COLOR") != "" || !isTerminal(out)))
	renderer := lipgloss.NewRenderer(out)
	if forceColor {
		renderer.SetColorProfile(termenv.TrueColor)
	}
	return &UI{
		out:   out,
		plain: plain,
		heading: renderer.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginTop(1),
		success: renderer.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		warning: renderer.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
		failure: renderer.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		muted:   renderer.NewStyle().Foreground(lipgloss.Color("245")),
		label:   renderer.NewStyle().Foreground(lipgloss.Color("245")),
		value:   renderer.NewStyle().Foreground(lipgloss.Color("252")),
		profile: renderer.NewStyle().Foreground(lipgloss.Color("81")).Bold(true),
		path:    renderer.NewStyle().Foreground(lipgloss.Color("111")),
		command: renderer.NewStyle().Foreground(lipgloss.Color("218")).Bold(true),
	}
}

func (u *UI) Plain() bool {
	return u.plain
}

func (u *UI) Printf(format string, args ...any) {
	fmt.Fprintf(u.out, format, args...)
}

func (u *UI) Println(args ...any) {
	fmt.Fprintln(u.out, args...)
}

func (u *UI) Success(message string, args ...any) {
	u.line(CheckOK, fmt.Sprintf(message, args...))
}

func (u *UI) Warning(message string, args ...any) {
	u.line(CheckWarn, fmt.Sprintf(message, args...))
}

func (u *UI) Failure(message string, args ...any) {
	u.line(CheckFail, fmt.Sprintf(message, args...))
}

func (u *UI) Heading(title string) {
	if u.plain {
		fmt.Fprintln(u.out, title)
		return
	}
	fmt.Fprintln(u.out, u.heading.Render(title))
}

func (u *UI) Section(title string) {
	if u.plain {
		fmt.Fprintf(u.out, "\n%s\n", title)
		return
	}
	fmt.Fprintln(u.out)
	fmt.Fprintln(u.out, u.heading.Render(title))
}

func (u *UI) KeyValue(key string, value any) {
	if u.plain {
		fmt.Fprintf(u.out, "%s: %v\n", key, value)
		return
	}
	fmt.Fprintf(u.out, "%s %s\n", u.label.Render(key+":"), u.value.Render(fmt.Sprint(value)))
}

func (u *UI) Bullet(text string, args ...any) {
	prefix := "-"
	if !u.plain {
		prefix = u.muted.Render("•")
	}
	fmt.Fprintf(u.out, "  %s %s\n", prefix, fmt.Sprintf(text, args...))
}

func (u *UI) Hint(command string) {
	if u.plain {
		fmt.Fprintf(u.out, "Hint: %s\n", command)
		return
	}
	fmt.Fprintf(u.out, "%s %s\n", u.muted.Render("Hint:"), u.command.Render(command))
}

func (u *UI) Check(state CheckState, label, detail string) {
	marker := string(state)
	switch state {
	case CheckOK:
		if !u.plain {
			marker = u.success.Render("✓")
		}
	case CheckWarn:
		if !u.plain {
			marker = u.warning.Render("!")
		}
	case CheckFail:
		if !u.plain {
			marker = u.failure.Render("✗")
		}
	}
	if u.plain {
		if detail == "" {
			fmt.Fprintf(u.out, "[%s] %s\n", state, label)
			return
		}
		fmt.Fprintf(u.out, "[%s] %s: %s\n", state, label, detail)
		return
	}
	if detail == "" {
		fmt.Fprintf(u.out, "%s %s\n", marker, u.value.Render(label))
		return
	}
	fmt.Fprintf(u.out, "%s %s %s\n", marker, u.value.Render(label), u.muted.Render(detail))
}

func (u *UI) ProfilesTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	if u.plain {
		return plainTable([]string{"Active", "Profile", "Git", "GitHub", "SSH"}, rows)
	}
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(u.muted).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		Headers("Active", "Profile", "Git", "GitHub", "SSH").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return u.label.Bold(true).Padding(0, 1)
			case col == 1:
				return u.profile.Padding(0, 1)
			default:
				return u.value.Padding(0, 1)
			}
		})
	return t.Render()
}

func (u *UI) ProfileName(name string) string {
	if u.plain {
		return name
	}
	return u.profile.Render(name)
}

func (u *UI) Path(path string) string {
	if u.plain {
		return path
	}
	return u.path.Render(path)
}

func (u *UI) line(state CheckState, message string) {
	u.Check(state, message, "")
}

func plainTable(headers []string, rows [][]string) string {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	var b strings.Builder
	writeRow := func(row []string) {
		for i, cell := range row {
			if i > 0 {
				b.WriteString("  ")
			}
			fmt.Fprintf(&b, "%-*s", widths[i], cell)
		}
		b.WriteByte('\n')
	}
	writeRow(headers)
	for i, width := range widths {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(strings.Repeat("-", width))
	}
	b.WriteByte('\n')
	for _, row := range rows {
		writeRow(row)
	}
	return strings.TrimRight(b.String(), "\n")
}

func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}
