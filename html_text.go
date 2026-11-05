// Package html_text reads an HTML mail body as text, as a mail client that ignores style sheets shows it: what the mail
// hides by an inline style or an attribute is left out, a table of data keeps its cells apart, a quote is marked as
// one, and a link keeps its target.
package html_text

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Text is the text of an HTML document. Mail carrying a copy for phones beside one for desktops hides one of them by an
// inline style and reveals it in a style sheet's media query, so reading it with the style sheet ignored shows each once.
func Text(source string) string {
	// Without scripting, as in a mail client, so that a noscript element's content is read as markup.
	// The parser fails only when its reader does, which a string's never does.
	document, err := html.ParseWithOptions(strings.NewReader(source), html.ParseOptionEnableScripting(false))
	if err != nil {
		return ""
	}

	if text := render(document, false); text != "" {
		return text
	}
	// A mail that hides all it says shows it by a style sheet: all of it is better read than nothing.
	return render(document, true)
}

func render(document *html.Node, reveal bool) string {
	w := &writer{reveal: reveal}
	w.children(document)
	return finish(w.builder.String())
}

type table struct {
	// layout says the table lays the mail out rather than holding data, so its cells need not be told apart.
	layout bool
	// rowHasText says the row has shown text, so that its next cell's text is set apart from it.
	rowHasText bool
}

func (t *table) separator() string {
	if t.layout {
		return " "
	}
	return " | "
}

type list struct {
	ordered bool
	count   int
}

// writer writes text as it is read, with no space at a line's start and none twice.
type writer struct {
	builder strings.Builder
	// reveal reads what the mail hides, too.
	reveal bool
	// breaks is how many line breaks are written before the next text: two leave a blank line. They wait for the text,
	// so that none ends the whole, and so that a link's target can follow its label on the label's line.
	breaks int
	// pendingSpace and pendingSeparator are written before the next word, unless a line break comes first.
	pendingSpace     bool
	pendingSeparator string
	// invisible says the text is hidden by visibility, which an element within can undo.
	invisible bool
	// pre is how deep the reading is in preformatted text, which is kept as it is; quote, in quotes, whose lines are
	// marked; links, in links; dataCell, in a data table's cells, which are read on their row's line, paragraphs and all.
	pre      int
	quote    int
	links    int
	dataCell int
	// afterMarker says a list item's marker was just written, which the item's first block does not break from.
	afterMarker bool
	// labelStart is where the text of the link being read begins, or -1 before any.
	labelStart int
	// lastTarget is the target of the link just read, and lastTargetText says whether text, not an image, said it: a mail
	// links an image and the text beside it to one place, which the second of them need not repeat.
	lastTarget     string
	lastTargetText bool
	tables         []*table
	lists          []*list
}

// skipped are the elements a mail client shows no text of.
var skipped = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Head: true, atom.Title: true, atom.Template: true, atom.Iframe: true,
	atom.Noframes: true, atom.Object: true, atom.Svg: true, atom.Math: true, atom.Input: true, atom.Select: true,
}

// blockBreaks are the line breaks around a block: two leave a blank line.
var blockBreaks = map[atom.Atom]int{
	atom.P: 2, atom.H1: 2, atom.H2: 2, atom.H3: 2, atom.H4: 2, atom.H5: 2, atom.H6: 2,
	atom.Div: 1, atom.Section: 1, atom.Article: 1, atom.Header: 1, atom.Footer: 1, atom.Nav: 1, atom.Aside: 1,
	atom.Main: 1, atom.Address: 1, atom.Center: 1, atom.Dl: 1, atom.Dt: 1, atom.Dd: 1, atom.Figure: 1,
	atom.Figcaption: 1, atom.Form: 1, atom.Fieldset: 1, atom.Legend: 1, atom.Caption: 1, atom.Details: 1,
	atom.Summary: 1,
}

func (w *writer) children(node *html.Node) {
	for child := range node.ChildNodes() {
		w.node(child)
	}
}

func (w *writer) node(node *html.Node) {
	switch node.Type {
	case html.TextNode:
		w.text(node.Data)
	case html.ElementNode:
		// An xml element is Office's settings in an Outlook mail, of no atom of its own.
		if skipped[node.DataAtom] || node.Data == "xml" {
			return
		}
		style := declarations(node)
		if !w.reveal && hides(node, style) {
			return
		}
		invisible := w.invisible
		switch style["visibility"] {
		case cssHidden, "collapse":
			w.invisible = true
		case "visible":
			w.invisible = false
		}
		w.element(node)
		w.invisible = invisible
	case html.DocumentNode:
		w.children(node)
	default:
	}
}

func (w *writer) element(node *html.Node) {
	switch node.DataAtom {
	case atom.Br:
		w.lineBreak()
	case atom.Img:
		// An image says nothing but as the label of a link.
	case atom.A:
		w.link(node)
	case atom.Hr:
		// A rule is all that marks where an Outlook reply's quoted original begins.
		w.breakLine(2)
		w.word("---")
		w.breakLine(2)
	case atom.Blockquote:
		w.afterMarker = false
		w.breakLine(2)
		w.quote++
		w.children(node)
		w.quote--
		w.breakLine(2)
	case atom.Pre:
		w.afterMarker = false
		w.breakLine(2)
		w.pre++
		w.children(node)
		w.pre--
		w.breakLine(2)
	case atom.Ul, atom.Ol:
		w.lists = append(w.lists, &list{ordered: node.DataAtom == atom.Ol})
		w.afterMarker = false
		w.breakLine(1)
		w.children(node)
		w.lists = w.lists[:len(w.lists)-1]
		w.breakLine(1)
	case atom.Li:
		w.breakLine(1)
		w.listMarker()
		w.children(node)
		w.afterMarker = false
		w.breakLine(1)
	case atom.Table:
		w.tables = append(w.tables, &table{layout: isLayoutTable(node)})
		w.afterMarker = false
		w.breakLine(1)
		w.children(node)
		w.tables = w.tables[:len(w.tables)-1]
		w.breakLine(1)
	case atom.Tr:
		if current := w.table(); current != nil {
			current.rowHasText = false
		}
		w.breakLine(1)
		w.children(node)
		w.breakLine(1)
	case atom.Td, atom.Th:
		current := w.table()
		if current != nil && current.rowHasText {
			w.pendingSeparator = current.separator()
		}
		data := current != nil && !current.layout
		if data {
			w.dataCell++
		}
		w.children(node)
		if data {
			w.dataCell--
		}
	default:
		if breaks, ok := blockBreaks[node.DataAtom]; ok {
			w.breakLine(breaks)
			w.children(node)
			w.breakLine(breaks)
			return
		}
		w.children(node)
	}
}

func (w *writer) table() *table {
	if len(w.tables) == 0 {
		return nil
	}
	return w.tables[len(w.tables)-1]
}

// listMarker marks a list item, indented by the lists it is in beyond the first.
func (w *writer) listMarker() {
	marker := "-"
	if len(w.lists) != 0 {
		if current := w.lists[len(w.lists)-1]; current.ordered {
			current.count++
			marker = strconv.Itoa(current.count) + "."
		}
		marker = strings.Repeat("  ", len(w.lists)-1) + marker
	}
	w.word(marker)
	w.pendingSpace = true
	w.afterMarker = true
}

// link is the link's label followed by its target, when the target is on the web or a mail address, is not what the
// label says already, and is not where the link just before went, the one of them an image and the other text; a link
// elsewhere -- a script, an anchor in the mail -- is its label alone.
func (w *writer) link(node *html.Node) {
	outerLabelStart := w.labelStart
	w.labelStart = -1
	w.links++
	w.children(node)
	w.links--
	labelStart := w.labelStart
	w.labelStart = outerLabelStart

	target := strings.TrimSpace(attribute(node, "href"))
	lowered := strings.ToLower(target)
	isMail := strings.HasPrefix(lowered, "mailto:")
	if (w.invisible && !w.reveal) || (!isMail && !strings.HasPrefix(lowered, "http://") && !strings.HasPrefix(lowered, "https://")) {
		return
	}

	label := ""
	if labelStart >= 0 {
		label = strings.TrimSpace(w.builder.String()[labelStart:])
	}
	textLabel := label != ""
	repeated := target == w.lastTarget && (!w.lastTargetText || !textLabel)
	if label == "" {
		if alt := imageAlt(node); alt != "" {
			w.links++
			w.text(alt)
			w.links--
			label = alt
		}
	}
	if repeated || label == target || (isMail && label == target[len("mailto:"):]) {
		w.lastTarget, w.lastTargetText = target, textLabel
		return
	}

	if label == "" {
		w.word("<" + target + ">")
	} else {
		// The target follows its label on the label's line, though the label ends a block, as a button's table does.
		breaks := w.breaks
		w.breaks = 0
		w.pendingSpace = true
		w.word("<" + target + ">")
		w.breaks = breaks
	}
	w.lastTarget, w.lastTargetText = target, textLabel
}

// text writes a text node's words, its blank space folded as a browser folds it, but in preformatted text.
func (w *writer) text(data string) {
	if w.pre > 0 {
		w.preformatted(data)
		return
	}
	if data == "" {
		return
	}

	if first, _ := utf8.DecodeRuneInString(data); isFoldedSpace(first) {
		w.pendingSpace = true
	}
	for index, field := range strings.FieldsFunc(data, isFoldedSpace) {
		if index > 0 {
			w.pendingSpace = true
		}
		w.word(field)
	}
	if last, _ := utf8.DecodeLastRuneInString(data); isFoldedSpace(last) {
		w.pendingSpace = true
	}
}

func (w *writer) word(word string) {
	word = strings.Map(dropInvisible, word)
	if word == "" || (w.invisible && !w.reveal) {
		return
	}

	w.spaceBefore()
	w.write(word)
}

// spaceBefore writes the separator or the space due before text on the line, if any is.
func (w *writer) spaceBefore() {
	if w.breaks != 0 || w.builder.Len() == 0 {
		return
	}
	switch {
	case w.pendingSeparator != "":
		w.builder.WriteString(w.pendingSeparator)
	case w.pendingSpace:
		w.builder.WriteByte(' ')
	}
}

func (w *writer) preformatted(data string) {
	if w.invisible && !w.reveal {
		return
	}
	for line := range strings.SplitAfterSeq(strings.ReplaceAll(data, "\r\n", "\n"), "\n") {
		if content := strings.TrimSuffix(line, "\n"); content != "" {
			w.spaceBefore()
			w.write(content)
		}
		if strings.HasSuffix(line, "\n") {
			w.lineBreak()
		}
	}
}

// write writes text after the line breaks due before it, a line in a quote marked as one, and notes that the rows of
// the tables it is in have text.
func (w *writer) write(text string) {
	atLineStart := w.breaks != 0 || w.builder.Len() == 0
	w.builder.WriteString(strings.Repeat("\n", w.breaks))
	if atLineStart && w.quote > 0 {
		w.builder.WriteString(strings.Repeat("> ", w.quote))
	}
	w.breaks = 0
	w.pendingSpace = false
	w.pendingSeparator = ""
	w.afterMarker = false

	if w.links > 0 && w.labelStart < 0 {
		w.labelStart = w.builder.Len()
	}
	w.builder.WriteString(text)
	for _, current := range w.tables {
		current.rowHasText = true
	}
	if w.links == 0 {
		w.lastTarget = ""
	}
}

// lineBreak breaks the line where it is, as a br does: a second in a row leaves a blank line. In a data table's cell,
// and right after a list item's marker, it is a space.
func (w *writer) lineBreak() {
	if w.dataCell > 0 || w.afterMarker {
		w.pendingSpace = true
		return
	}
	if w.builder.Len() != 0 {
		w.breaks++
	}
	w.pendingSpace = false
	w.pendingSeparator = ""
}

// breakLine ends the line unless it has ended, with at least count line breaks: two leave a blank line. In a data
// table's cell, and right after a list item's marker, it is a space.
func (w *writer) breakLine(count int) {
	if w.dataCell > 0 || w.afterMarker {
		w.pendingSpace = true
		return
	}
	if w.builder.Len() != 0 {
		w.breaks = max(w.breaks, count)
	}
	w.pendingSpace = false
	w.pendingSeparator = ""
}

// cssHidden is the keyword with which visibility hides an element, and overflow clips its content.
const cssHidden = "hidden"

// hides says the element and all within it are hidden, by the hidden attribute or by its own style: not displayed,
// transparent, or clipped to no height. A font size of zero does not hide: layouts set it to drop the space between
// columns, whose content sets its own. Visibility is not of these, as an element within can undo it.
func hides(node *html.Node, style map[string]string) bool {
	for _, attr := range node.Attr {
		if attr.Key == "hidden" {
			return true
		}
	}

	switch {
	case style["display"] == "none", isZero(style["opacity"]):
		return true
	case style["overflow"] == cssHidden && (isZero(style["max-height"]) || isZero(style["height"])):
		return true
	default:
		return false
	}
}

// declarations are the element's inline style, by property, lower case and without !important.
func declarations(node *html.Node) map[string]string {
	style := attribute(node, "style")
	if style == "" {
		return nil
	}

	parsed := map[string]string{}
	for declaration := range strings.SplitSeq(style, ";") {
		name, value, found := strings.Cut(declaration, ":")
		if !found {
			continue
		}
		if important := strings.Index(value, "!"); important >= 0 {
			value = value[:important]
		}
		parsed[strings.ToLower(strings.TrimSpace(name))] = strings.ToLower(strings.TrimSpace(value))
	}
	return parsed
}

// isZero says a length or number is zero, in whatever unit.
func isZero(value string) bool {
	number := strings.TrimRight(value, "abcdefghijklmnopqrstuvwxyz%")
	if number == "" {
		return false
	}
	parsed, err := strconv.ParseFloat(number, 64)
	return err == nil && parsed == 0
}

// isLayoutTable says a table lays the mail out: one marked so, or one holding another table.
func isLayoutTable(node *html.Node) bool {
	switch strings.ToLower(strings.TrimSpace(attribute(node, "role"))) {
	case "presentation", "none":
		return true
	}
	for descendant := range node.Descendants() {
		if descendant.Type == html.ElementNode && descendant.DataAtom == atom.Table {
			return true
		}
	}
	return false
}

// imageAlt is the text of the first image in the node that has any.
func imageAlt(node *html.Node) string {
	for descendant := range node.Descendants() {
		if descendant.Type == html.ElementNode && descendant.DataAtom == atom.Img {
			if alt := strings.TrimSpace(attribute(descendant, "alt")); alt != "" {
				return alt
			}
		}
	}
	return ""
}

func attribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// isFoldedSpace says a character is blank space a browser folds into one; a no-break space is read as one too.
func isFoldedSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\f', '\r', '\u00a0':
		return true
	default:
		return false
	}
}

// dropInvisible drops the characters that show nothing, with which mail pads the preview text an inbox shows: a soft
// hyphen, a combining grapheme joiner, a zero-width space and non-joiner, a word joiner, and a byte order mark. The
// zero-width joiner is kept, which joins an emoji of several.
func dropInvisible(r rune) rune {
	switch r {
	case '\u00ad', '\u034f', '\u200b', '\u200c', '\u2060', '\ufeff':
		return -1
	default:
		return r
	}
}

var blankLineRunPattern = regexp.MustCompile(`\n{3,}`)

// finish takes the space from the ends of the lines, and leaves no more than one blank line in a row.
func finish(text string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRightFunc(line, unicode.IsSpace)
	}
	return strings.Trim(blankLineRunPattern.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"), "\n")
}
