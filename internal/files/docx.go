package files

import (
	"archive/zip"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"path"
	"strings"
)

const docxCSP = "default-src 'none'; img-src data:; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'self'"

type relationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type relationships struct {
	Items []relationship `xml:"Relationship"`
}

// docxPreviewHTML renders a self-contained HTML preview from a .docx file.
// It intentionally covers the common pi-web use case: paragraphs, text, line
// breaks, tabs, and inline images embedded as data URIs.
func docxPreviewHTML(filePath string) (string, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer zr.Close()

	entries := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		entries[f.Name] = f
	}
	doc, ok := entries["word/document.xml"]
	if !ok {
		return "", fmt.Errorf("invalid docx: missing word/document.xml")
	}

	rels := map[string]string{}
	if relFile, ok := entries["word/_rels/document.xml.rels"]; ok {
		rels, err = parseDocxRels(relFile)
		if err != nil {
			return "", err
		}
	}
	body, err := renderDocxBody(doc, rels, entries)
	if err != nil {
		return "", err
	}
	return wrapDocxPreviewHTML(body, path.Base(filePath)), nil
}

func parseDocxRels(f *zip.File) (map[string]string, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var rels relationships
	if err := xml.NewDecoder(rc).Decode(&rels); err != nil {
		return nil, fmt.Errorf("parse docx rels: %w", err)
	}
	out := make(map[string]string, len(rels.Items))
	for _, r := range rels.Items {
		out[r.ID] = r.Target
	}
	return out, nil
}

func renderDocxBody(doc *zip.File, rels map[string]string, entries map[string]*zip.File) (string, error) {
	rc, err := doc.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var paragraphs []string
	var paragraph strings.Builder
	inParagraph := false
	readingText := false

	flush := func() {
		if paragraph.Len() == 0 {
			return
		}
		paragraphs = append(paragraphs, "<p>"+paragraph.String()+"</p>")
		paragraph.Reset()
	}

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("parse document.xml: %w", err)
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				flush()
				inParagraph = true
			case "t":
				if inParagraph {
					readingText = true
				}
			case "tab":
				if inParagraph {
					paragraph.WriteString("\t")
				}
			case "br":
				if inParagraph {
					paragraph.WriteString("<br>")
				}
			case "blip":
				if inParagraph {
					if rid := xmlAttr(t, "embed"); rid != "" {
						if img := docxImageDataURI(rid, rels, entries); img != "" {
							paragraph.WriteString(`<img src="` + img + `" alt="">`)
						}
					}
				}
			}
		case xml.CharData:
			if readingText {
				paragraph.WriteString(html.EscapeString(string(t)))
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				readingText = false
			case "p":
				flush()
				inParagraph = false
			}
		}
	}
	flush()
	return strings.Join(paragraphs, ""), nil
}

func xmlAttr(start xml.StartElement, local string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

func docxImageDataURI(rid string, rels map[string]string, entries map[string]*zip.File) string {
	target, ok := rels[rid]
	if !ok {
		return ""
	}
	if strings.HasPrefix(target, "/") {
		target = strings.TrimPrefix(target, "/")
	} else {
		target = path.Clean(path.Join("word", target))
	}
	f, ok := entries[target]
	if !ok {
		return ""
	}
	rc, err := f.Open()
	if err != nil {
		return ""
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return ""
	}
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(target)), ".")
	mimeType := "image/png"
	switch ext {
	case "jpg", "jpeg":
		mimeType = "image/jpeg"
	case "gif":
		mimeType = "image/gif"
	case "webp":
		mimeType = "image/webp"
	case "svg":
		mimeType = "image/svg+xml"
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func wrapDocxPreviewHTML(bodyHTML, fileName string) string {
	return `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
  html, body { margin: 0; min-height: 100%; background: #eef1f5; color: #171717; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; padding: 28px; }
  main { box-sizing: border-box; max-width: 840px; min-height: calc(100vh - 56px); margin: 0 auto; padding: 56px 64px; background: #fff; box-shadow: 0 8px 28px rgba(15,23,42,.14); }
  .file-title { margin: 0 0 28px; padding-bottom: 10px; border-bottom: 1px solid #e5e7eb; color: #6b7280; font: 12px ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; word-break: break-word; }
  h1, h2, h3, h4, h5, h6 { line-height: 1.3; margin: 1.1em 0 .45em; color: #111827; }
  p { margin: .65em 0; line-height: 1.7; }
  table { border-collapse: collapse; max-width: 100%; margin: 1em 0; }
  th, td { border: 1px solid #d1d5db; padding: 6px 9px; vertical-align: top; }
  img { max-width: 100%; height: auto; }
  @media (max-width: 720px) { body { padding: 0; background: #fff; } main { min-height: 100vh; padding: 28px 22px; box-shadow: none; } }
</style>
</head>
<body>
<main>
<div class="file-title">` + html.EscapeString(fileName) + `</div>
` + bodyHTML + `
</main>
</body>
</html>`
}
