# html_text

Reads an HTML mail body as text, as a mail client that ignores style sheets shows it: what the mail hides by an inline style or an attribute is left out, a table of data keeps its cells apart, a quote is marked as one, and a link keeps its target.

```go
import "github.com/altshiftab/html_text"

text := html_text.Text(body)
```

Its one dependency is `golang.org/x/net/html`.
