/*
 * markdown.go - Copyright (c) 2023-2026 - Olivier Poncet
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package assistant

import (
    "net/url"
    "strings"
    "unicode"
)

// ---------------------------------------------------------------------------
// Markdown declaration
// ---------------------------------------------------------------------------

type Markdown struct {
    baseLevel int
}

type markdownLines struct {
    lines []string
    index int
}

const (
    maxHeadingLevel = 6
    fence           = "```"
)

// ---------------------------------------------------------------------------
// Markdown implementation
// ---------------------------------------------------------------------------

func NewMarkdown(baseLevel int) *Markdown {
    return &Markdown{baseLevel: baseLevel}
}

func (m *Markdown) Parse(text string) []Block {
    source := &markdownLines{lines: strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")}
    blocks := []Block{}
    paragraph := []string{}
    for source.index < len(source.lines) {
        line := source.lines[source.index]
        trimmed := strings.TrimSpace(line)
        block, consumed := m.block(source, trimmed)
        if !consumed {
            if trimmed != "" {
                paragraph = append(paragraph, trimmed)
            }
            source.index++
        }
        if consumed || (trimmed == "") {
            blocks = m.paragraph(blocks, paragraph)
            paragraph = []string{}
        }
        if consumed {
            blocks = append(blocks, block)
        }
    }
    blocks = m.paragraph(blocks, paragraph)
    return m.number(blocks)
}

func (m *Markdown) block(source *markdownLines, trimmed string) (Block, bool) {
    switch {
        case strings.HasPrefix(trimmed, fence):
            return m.code(source), true
        case m.heading(trimmed) > 0:
            level := m.heading(trimmed)
            source.index++
            return Block{Kind: BlockHeading, Level: level, Text: m.Inlines(strings.TrimSpace(trimmed[level:]))}, true
        case m.rule(trimmed):
            source.index++
            return Block{Kind: BlockRule}, true
        case strings.HasPrefix(trimmed, ">"):
            return m.quote(source), true
        case m.item(trimmed) != "":
            return m.list(source), true
        case m.table(source):
            return m.rows(source), true
    }
    return Block{}, false
}

func (m *Markdown) paragraph(blocks []Block, lines []string) []Block {
    if len(lines) == 0 {
        return blocks
    }
    return append(blocks, Block{Kind: BlockParagraph, Text: m.Inlines(strings.Join(lines, " "))})
}

func (m *Markdown) number(blocks []Block) []Block {
    tables, previous := 0, m.baseLevel - 1
    for i, block := range blocks {
        switch block.Kind {
            case BlockTable:
                tables++
                blocks[i].Number = tables
            case BlockHeading:
                level := min(m.baseLevel + block.Level - 1, previous + 1, maxHeadingLevel)
                blocks[i].Level, previous = level, level
        }
    }
    return blocks
}

func (m *Markdown) heading(trimmed string) int {
    level := 0
    for (level < len(trimmed)) && (trimmed[level] == '#') {
        level++
    }
    if (level == 0) || (level > maxHeadingLevel) || (level >= len(trimmed)) || (trimmed[level] != ' ') {
        return 0
    }
    return level
}

func (m *Markdown) rule(trimmed string) bool {
    compact := strings.ReplaceAll(trimmed, " ", "")
    if len(compact) < 3 {
        return false
    }
    for _, mark := range []string{"-", "*", "_"} {
        if strings.Trim(compact, mark) == "" {
            return true
        }
    }
    return false
}

func (m *Markdown) code(source *markdownLines) Block {
    lines := []string{}
    for source.index++; source.index < len(source.lines); source.index++ {
        if strings.HasPrefix(strings.TrimSpace(source.lines[source.index]), fence) {
            source.index++
            break
        }
        lines = append(lines, source.lines[source.index])
    }
    return Block{Kind: BlockCode, Code: strings.Join(lines, "\n")}
}

func (m *Markdown) quote(source *markdownLines) Block {
    lines := []string{}
    for ; source.index < len(source.lines); source.index++ {
        trimmed := strings.TrimSpace(source.lines[source.index])
        if !strings.HasPrefix(trimmed, ">") {
            break
        }
        if content := strings.TrimSpace(strings.TrimPrefix(trimmed, ">")); content != "" {
            lines = append(lines, content)
        }
    }
    return Block{Kind: BlockQuote, Text: m.Inlines(strings.Join(lines, " "))}
}

func (m *Markdown) item(trimmed string) string {
    for _, bullet := range []string{"- ", "* ", "+ "} {
        if strings.HasPrefix(trimmed, bullet) && !m.rule(trimmed) {
            return strings.TrimSpace(trimmed[len(bullet):])
        }
    }
    digits := 0
    for (digits < len(trimmed)) && (digits < 9) && (trimmed[digits] >= '0') && (trimmed[digits] <= '9') {
        digits++
    }
    if (digits > 0) && (digits + 1 < len(trimmed)) && ((trimmed[digits] == '.') || (trimmed[digits] == ')')) && (trimmed[digits + 1] == ' ') {
        return strings.TrimSpace(trimmed[digits + 2:])
    }
    return ""
}

func (m *Markdown) list(source *markdownLines) Block {
    first := strings.TrimSpace(source.lines[source.index])
    block := Block{Kind: BlockList, Ordered: (first[0] >= '0') && (first[0] <= '9')}
    items := []string{}
    for ; source.index < len(source.lines); source.index++ {
        line := source.lines[source.index]
        trimmed := strings.TrimSpace(line)
        if content := m.item(trimmed); content != "" {
            items = append(items, content)
            continue
        }
        if (trimmed == "") || !unicode.IsSpace(rune(line[0])) || (len(items) == 0) {
            break
        }
        items[len(items) - 1] += " " + trimmed
    }
    for _, item := range items {
        block.Items = append(block.Items, m.Inlines(item))
    }
    return block
}

func (m *Markdown) table(source *markdownLines) bool {
    if (source.index + 1) >= len(source.lines) {
        return false
    }
    header, separator := strings.TrimSpace(source.lines[source.index]), strings.TrimSpace(source.lines[source.index + 1])
    if !strings.Contains(header, "|") || !strings.Contains(separator, "-") {
        return false
    }
    for _, cell := range m.cells(separator) {
        if (strings.Trim(cell, ":-") != "") || !strings.Contains(cell, "-") {
            return false
        }
    }
    return len(m.cells(header)) == len(m.cells(separator))
}

func (m *Markdown) rows(source *markdownLines) Block {
    block := Block{Kind: BlockTable}
    header := m.cells(strings.TrimSpace(source.lines[source.index]))
    for _, cell := range header {
        block.Header = append(block.Header, m.Inlines(cell))
    }
    for source.index += 2; source.index < len(source.lines); source.index++ {
        trimmed := strings.TrimSpace(source.lines[source.index])
        if !strings.Contains(trimmed, "|") {
            break
        }
        row := [][]Inline{}
        cells := m.cells(trimmed)
        for i := range header {
            cell := ""
            if i < len(cells) {
                cell = cells[i]
            }
            row = append(row, m.Inlines(cell))
        }
        block.Rows = append(block.Rows, row)
    }
    return block
}

func (m *Markdown) cells(line string) []string {
    line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
    cells := []string{}
    for _, cell := range strings.Split(line, "|") {
        cells = append(cells, strings.TrimSpace(cell))
    }
    return cells
}

func (m *Markdown) Inlines(text string) []Inline {
    inlines := []Inline{}
    plain := strings.Builder{}
    for i := 0; i < len(text); {
        inline, width := m.inline(text[i:])
        if width == 0 {
            plain.WriteByte(text[i])
            i++
            continue
        }
        if plain.Len() > 0 {
            inlines = append(inlines, Inline{Kind: InlineText, Text: plain.String()})
            plain.Reset()
        }
        inlines = append(inlines, inline)
        i += width
    }
    if plain.Len() > 0 {
        inlines = append(inlines, Inline{Kind: InlineText, Text: plain.String()})
    }
    return inlines
}

func (m *Markdown) inline(text string) (Inline, int) {
    switch {
        case strings.HasPrefix(text, "`"):
            return m.span(text, "`", InlineCode)
        case strings.HasPrefix(text, "**"):
            return m.span(text, "**", InlineStrong)
        case strings.HasPrefix(text, "__"):
            return m.span(text, "__", InlineStrong)
        case strings.HasPrefix(text, "*") && (len(text) > 1) && (text[1] != ' '):
            return m.span(text, "*", InlineEm)
        case strings.HasPrefix(text, "["):
            return m.link(text)
    }
    return Inline{}, 0
}

func (m *Markdown) span(text string, mark string, kind string) (Inline, int) {
    end := strings.Index(text[len(mark):], mark)
    if end <= 0 {
        return Inline{}, 0
    }
    content := text[len(mark):len(mark) + end]
    if (kind != InlineCode) && (strings.TrimSpace(content) != content) {
        return Inline{}, 0
    }
    return Inline{Kind: kind, Text: m.plain(content, kind)}, (2 * len(mark)) + end
}

func (m *Markdown) plain(content string, kind string) string {
    if kind == InlineCode {
        return content
    }
    return strings.NewReplacer("**", "", "__", "", "`", "").Replace(content)
}

func (m *Markdown) link(text string) (Inline, int) {
    label := strings.Index(text, "](")
    if (label <= 1) || strings.Contains(text[1:label], "]") {
        return Inline{}, 0
    }
    end := strings.Index(text[label + 2:], ")")
    if end <= 0 {
        return Inline{}, 0
    }
    target := strings.TrimSpace(text[label + 2:label + 2 + end])
    width := label + 3 + end
    caption := m.plain(text[1:label], InlineText)
    if m.download(target) {
        return Inline{Kind: InlineLink, Text: caption, Href: target, Download: true}, width
    }
    if !m.safe(target) {
        return Inline{Kind: InlineText, Text: caption}, width
    }
    return Inline{Kind: InlineLink, Text: caption, Href: target}, width
}

func (m *Markdown) download(target string) bool {
    id, found := strings.CutPrefix(target, DownloadsPath)
    if !found || (len(id) != downloadIDLength) {
        return false
    }
    for _, letter := range []byte(id) {
        if ((letter < '0') || (letter > '9')) && ((letter < 'a') || (letter > 'f')) {
            return false
        }
    }
    return true
}

func (m *Markdown) safe(target string) bool {
    parsed, err := url.Parse(target)
    return (err == nil) && ((parsed.Scheme == "https") || (parsed.Scheme == "http")) && (parsed.Host != "") && (parsed.User == nil)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
