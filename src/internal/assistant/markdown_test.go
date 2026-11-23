/*
 * markdown_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "reflect"
    "testing"
)

// ---------------------------------------------------------------------------
// Markdown tests
// ---------------------------------------------------------------------------

func TestMarkdownParsesTablesListsAndHeadings(t *testing.T) {
    text := "# Soldes\n\nVoici **vos** soldes :\n\n| Compteur | Solde |\n| --- | ---: |\n| CP | 12,5 |\n| RTT | `3` |\n\n- un\n- deux\n  suite\n\n1. premier\n2) second\n\n### Détail\n\n> cité\n\n```\n<b>code</b>\n```\n\n---"
    blocks := NewMarkdown(3).Parse(text)
    kinds := []string{}
    for _, block := range blocks {
        kinds = append(kinds, block.Kind)
    }
    want := []string{BlockHeading, BlockParagraph, BlockTable, BlockList, BlockList, BlockHeading, BlockQuote, BlockCode, BlockRule}
    if !reflect.DeepEqual(kinds, want) {
        t.Fatalf("got %v", kinds)
    }
    if (blocks[0].Level != 3) || (blocks[5].Level != 4) {
        t.Fatalf("the headings must start at the base level and never skip one: %d %d", blocks[0].Level, blocks[5].Level)
    }
    table := blocks[2]
    if (len(table.Header) != 2) || (len(table.Rows) != 2) || (table.Rows[1][1][0].Kind != InlineCode) || (table.Number != 1) {
        t.Fatalf("got %+v", table)
    }
    if (len(blocks[3].Items) != 2) || blocks[3].Ordered || !blocks[4].Ordered || (blocks[3].Items[1][0].Text != "deux suite") {
        t.Fatalf("got %+v %+v", blocks[3], blocks[4])
    }
    if blocks[7].Code != "<b>code</b>" {
        t.Fatalf("got %q", blocks[7].Code)
    }
    if strong := blocks[1].Text[1]; (strong.Kind != InlineStrong) || (strong.Text != "vos") {
        t.Fatalf("got %+v", blocks[1].Text)
    }
}

func TestMarkdownKeepsOnlyWebLinks(t *testing.T) {
    inlines := NewMarkdown(3).Inlines("[ok](https://service-public.fr/x) [ko](javascript:alert(1)) [local](/dashboard/) *em* a * b")
    if (inlines[0].Kind != InlineLink) || (inlines[0].Href != "https://service-public.fr/x") {
        t.Fatalf("got %+v", inlines[0])
    }
    for _, inline := range inlines[1:] {
        if inline.Kind == InlineLink {
            t.Fatalf("an unsafe link was kept: %+v", inline)
        }
    }
    found := false
    for _, inline := range inlines {
        found = found || ((inline.Kind == InlineEm) && (inline.Text == "em"))
    }
    if !found {
        t.Fatalf("got %+v", inlines)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
