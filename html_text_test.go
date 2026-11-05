package html_text

import "testing"

func TestText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{name: "empty"},
		{
			name:   "blank space folded",
			source: "  one\n  two\t three  ",
			want:   "one two three",
		},
		{
			name:   "paragraphs and breaks",
			source: "<p>one</p><p>two<br>three</p><div>four</div>",
			want:   "one\n\ntwo\nthree\n\nfour",
		},
		{
			name:   "space between inline elements kept, and none made",
			source: "<b>VIKTIGT!</b> Postadress <i>här</i>.<span>a</span><span>b</span>",
			want:   "VIKTIGT! Postadress här.ab",
		},
		{
			name:   "entities",
			source: "caf&eacute; &lt;b&gt; &#8364;&nbsp;5",
			want:   "café <b> € 5",
		},
		{
			name:   "head, style and script dropped",
			source: "<html><head><title>T</title><style>p { color: red }</style></head><body>visible<script>alert(1)</script></body></html>",
			want:   "visible",
		},
		{
			name:   "an inline display none hides all within it",
			source: `<div>shown</div><div style="display:none"><table><tr><td><div>hidden <b>deep</b></div></td></tr></table></div><div>after</div>`,
			want:   "shown\nafter",
		},
		{
			name: "every way an inline style or attribute hides",
			source: `<p hidden>a</p><p style="DISPLAY : NONE ! important">b</p><p style="visibility:hidden">c</p>` +
				`<p style="opacity:0">d</p><p style="max-height:0px;overflow:hidden">e</p><p style="height:0;overflow:hidden">f</p><p>g</p>`,
			want: "g",
		},
		{
			name:   "styles that do not hide",
			source: `<div style="font-size:0"><span style="font-size:14px">a</span></div><p style="mso-hide:all">b</p><p style="display:none;display:block">c</p><p style="max-height:0">d</p>`,
			want:   "a\n\nb\n\nc\n\nd",
		},
		{
			name: "a copy for phones hidden inline and shown by a media query is read once",
			source: `<style>@media (max-width:699px){.desktop{display:none!important}.mobile{display:block!important}}</style>` +
				`<table role="presentation"><tr style="display:none" class="mobile"><td>Nov 5, 2026 , 2:04 PM</td></tr>` +
				`<tr><td class="desktop">Nov 5, 2026 2:04 PM</td></tr></table>` +
				`<div class="desktop">Need help?</div><div class="mobile" style="display:none">Need help?</div>`,
			want: "Nov 5, 2026 2:04 PM\nNeed help?",
		},
		{
			name: "a table of data keeps its cells apart",
			source: "<table><tr><th>ID</th><th>Title</th><th>Location</th></tr>" +
				"<tr><td>17222</td><td>Mechanical engineer</td><td>Södertälje</td></tr></table>",
			want: "ID | Title | Location\n17222 | Mechanical engineer | Södertälje",
		},
		{
			name:   "empty cells are not set apart",
			source: "<table><tr><td></td><td>a</td><td> </td><td><img src=x.png></td><td>b</td></tr></table>",
			want:   "a | b",
		},
		{
			name:   "a layout table's cells are set apart by a space",
			source: `<table role="presentation"><tr><td>Total</td><td>SEK 145.00</td></tr></table>`,
			want:   "Total SEK 145.00",
		},
		{
			name:   "a table holding a table lays out",
			source: "<table><tr><td><b>Trip</b></td><td><table><tr><td>fare</td></tr></table></td></tr></table>",
			want:   "Trip\nfare",
		},
		{
			name:   "a data cell's blocks stay on its row's line",
			source: "<table><tr><td>a</td><td><div>b</div><p>c<br>d</p></td></tr></table>",
			want:   "a | b c d",
		},
		{
			name: "an Outlook table, its cells paragraphs",
			source: `<table border=1><tr><td><p class=MsoNormal>ID<o:p></o:p></p></td><td><p class=MsoNormal>Title</p></td></tr>` +
				`<tr><td><p class=MsoNormal>1</p></td><td><p class=MsoNormal>X</p></td></tr></table>`,
			want: "ID | Title\n1 | X",
		},
		{
			name:   "a list item's first block follows its marker",
			source: "<ul><li><p>one</p></li><li><p>two</p><p>more</p></li></ul><ol><li><div>a</div></li><li></li><li>b</li></ol>",
			want:   "- one\n\n- two\n\nmore\n\n1. a\n2.\n3. b",
		},
		{
			name:   "a link that says its target says it once in a cell and in a quote",
			source: `<table><tr><td>a</td><td><a href="https://x.test">https://x.test</a></td></tr></table><blockquote><a href="https://x.test/q">https://x.test/q</a></blockquote>`,
			want:   "a | https://x.test\n\n> https://x.test/q",
		},
		{
			name:   "a hidden link does not take the target from the next",
			source: `<span style="visibility:hidden"><a href="https://x.test/o">h</a></span><a href="https://x.test/o">Visible</a>`,
			want:   "Visible <https://x.test/o>",
		},
		{
			name: "text links to one place in rows give the target each time, an image's and its text's once",
			source: `<table role="presentation"><tr><td><a href="https://x.test">A</a></td></tr><tr><td><a href="https://x.test">B</a></td></tr>` +
				`<tr><td><a href="https://x.test/p"><img src=p.png></a></td></tr><tr><td><a href="https://x.test/p">Title</a></td></tr></table>`,
			want: "A <https://x.test>\nB <https://x.test>\n<https://x.test/p>\nTitle",
		},
		{
			name:   "preformatted text in a data cell keeps to its row, set apart",
			source: "<table><tr><td>a</td><td><pre>b\nc</pre></td><td>d</td></tr></table>",
			want:   "a | b c | d",
		},
		{
			name:   "an item's first block of its own kind breaks from its marker",
			source: "<ul><li><pre>code</pre></li><li><blockquote>q</blockquote></li></ul>",
			want:   "-\n\ncode\n\n-\n\n> q",
		},
		{
			name:   "a list in a list is indented",
			source: "<ul><li>a<ul><li>b</li><li><ol><li>c</li></ol></li></ul></li><li>d</li></ul>",
			want:   "- a\n  - b\n  -\n    1. c\n- d",
		},
		{
			name:   "an emoji of several keeps its joiners",
			source: "\U0001F468\u200d\U0001F469\u200d\U0001F467 family",
			want:   "\U0001F468\u200d\U0001F469\u200d\U0001F467 family",
		},
		{
			name:   "lists",
			source: "<ul><li>a</li><li>b</li></ul><ol><li>one</li><li>two</li></ol>",
			want:   "- a\n- b\n1. one\n2. two",
		},
		{
			name:   "a link keeps its target",
			source: `<p>Confirm <a class="b" href="https://example.com/c?a=1&amp;b=2">here</a>.</p>`,
			want:   "Confirm here <https://example.com/c?a=1&b=2>.",
		},
		{
			name:   "a link that says its target says it once",
			source: `<a href="https://example.com">https://example.com</a> <a href='mailto:a@example.com'>a@example.com</a>`,
			want:   "https://example.com a@example.com",
		},
		{
			name:   "a link of an image is its alt text and target",
			source: `<a href="https://example.com/fb"><img src="fb.png" alt="Facebook"></a> <a href="https://example.com/x"><img src="logo.png"></a>`,
			want:   "Facebook <https://example.com/fb> <https://example.com/x>",
		},
		{
			name:   "a link elsewhere than the web keeps its text alone",
			source: `<a href="javascript:alert(1)">click</a> <a href="#top">top</a> <a>bare</a>`,
			want:   "click top bare",
		},
		{
			name:   "a button's link wrapping a table",
			source: `<a href="https://example.com/help"><table role="presentation"><tr><td><img src=i.png></td><td>Contact support</td></tr></table></a>`,
			want:   "Contact support <https://example.com/help>",
		},
		{
			name:   "preformatted text keeps its indentation",
			source: "<p>x</p><pre>  a\n    b</pre><p>y</p>",
			want:   "x\n\n  a\n    b\n\ny",
		},
		{
			name:   "noscript is read, as a mail client runs no script",
			source: "<noscript><b>no</b> script</noscript>",
			want:   "no script",
		},
		{
			name:   "the invisible padding of a preview text is dropped",
			source: "Hello&zwnj; &#847; &#847;&zwnj; &#173;world&#65279;",
			want:   "Hello world",
		},
		{
			name:   "unclosed elements",
			source: "<p>one<p>two<div>three",
			want:   "one\n\ntwo\n\nthree",
		},
		{
			name:   "conditional comments for Outlook are dropped",
			source: "<!--[if mso]><table><tr><td>outlook</td></tr></table><![endif]--><!--[if !mso]><!--><div>others</div><!--<![endif]-->",
			want:   "others",
		},
		{
			name:   "blank lines collapsed",
			source: "a<br><br><br><br>b",
			want:   "a\n\nb",
		},
		{
			name:   "no-break spacer cells are empty",
			source: "<table><tr><td>&nbsp;</td><td>a</td><td>&nbsp;&nbsp;</td><td>b</td></tr></table>",
			want:   "a | b",
		},
		{
			name:   "an amount's no-break spaces and a word's soft hyphen",
			source: "Att betala: 1&nbsp;250&nbsp;kr f&ouml;r&shy;s&auml;kring",
			want:   "Att betala: 1 250 kr försäkring",
		},
		{
			name:   "a role none table lays out",
			source: `<table role="none"><tr><td>Total</td><td>145 kr</td></tr></table>`,
			want:   "Total 145 kr",
		},
		{
			name:   "an image's link and then its text's to one place give the target once",
			source: `<a href="https://x.test/o"><img src="a.png"></a> <a href="https://x.test/o">View order</a>`,
			want:   "<https://x.test/o> View order",
		},
		{
			name:   "a text's link and then its image's to one place give the target once",
			source: `<a href="https://x.test/o">View order</a><a href="https://x.test/o"><img src="a.png"></a>`,
			want:   "View order <https://x.test/o>",
		},
		{
			name:   "links to one place with text between give the target each time",
			source: `<a href="https://x.test/o">Read</a> more text <a href="https://x.test/o">Read more</a>`,
			want:   "Read <https://x.test/o> more text Read more <https://x.test/o>",
		},
		{
			name: "a quote in a quote is marked twice",
			source: `<div>Thanks!</div><div class="gmail_quote"><div>On Mon, A wrote:</div>` +
				`<blockquote class="gmail_quote"><div>Sure</div><blockquote type="cite">Can you?<br>Today?</blockquote></blockquote></div>`,
			want: "Thanks!\nOn Mon, A wrote:\n\n> Sure\n\n> > Can you?\n> > Today?",
		},
		{
			name: "an Outlook reply marks its original with a rule",
			source: `<!--[if gte mso 9]><xml><o:OfficeDocumentSettings><o:AllowPNG/></o:OfficeDocumentSettings></xml><![endif]-->` +
				`<p class="MsoNormal">Hej<o:p></o:p></p><div id="divRplyFwdMsg"><hr><p><b>From:</b> A<br><b>Sent:</b> Monday</p></div><p>Original</p>`,
			want: "Hej\n\n---\n\nFrom: A\nSent: Monday\n\nOriginal",
		},
		{
			name:   "Office settings, a form's fields and a choice are not text",
			source: "<xml><o:Settings>junk</o:Settings></xml><select><option>One</option></select><input value=x>after",
			want:   "after",
		},
		{
			name: "an Outlook button and the others' link are read once",
			source: `<!--[if mso]><v:roundrect href="https://x.test/b"><center>Buy</center></v:roundrect><![endif]-->` +
				`<!--[if !mso]><!--><a href="https://x.test/b">Buy</a><!--<![endif]-->`,
			want: "Buy <https://x.test/b>",
		},
		{
			name:   "center and font",
			source: "<center><font size=2>a</font></center><center>b</center>",
			want:   "a\nb",
		},
		{
			name:   "visibility hidden is undone within",
			source: `<div style="visibility:hidden">a<span style="visibility:visible">b</span></div>c`,
			want:   "b\nc",
		},
		{
			name:   "a mail that hides all it says is read whole",
			source: `<style>.show{display:block!important}</style><div class="show" style="display:none">Only copy</div>`,
			want:   "Only copy",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Text(testCase.source); got != testCase.want {
				t.Errorf("text = %q, want %q", got, testCase.want)
			}
		})
	}
}
