package feeds

import (
	"bytes"
	"encoding/xml"
)

type opmlOutline struct {
	Text     string `xml:"text,attr"`
	Title    string `xml:"title,attr"`
	Type     string `xml:"type,attr"`
	XMLURL   string `xml:"xmlUrl,attr"`
	HTMLURL  string `xml:"htmlUrl,attr"`
	Children []opmlOutline `xml:"outline"`
}

type opmlDoc struct {
	XMLName xml.Name `xml:"opml"`
	Body    struct {
		Outlines []opmlOutline `xml:"outline"`
	} `xml:"body"`
}

// ImportOPML parses an OPML document, creating folders and feeds as structured
// (flat rss outlines land uncategorized). Returns the number of feeds added.
// Duplicate/conflicting feeds are skipped without failing the import.
func (r *Repo) ImportOPML(data []byte) (int, error) {
	var doc opmlDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return 0, err
	}
	added := 0
	for _, o := range doc.Body.Outlines {
		if o.XMLURL != "" {
			if _, err := r.addFeedFromOPML(o, nil); err == nil {
				added++
			}
			continue
		}
		name := outlineName(o)
		var folderID *int
		if name != "" {
			if folder, err := r.ensureFolder(name); err == nil {
				folderID = &folder.ID
			}
		}
		for _, c := range o.Children {
			if c.XMLURL == "" {
				continue
			}
			if _, err := r.addFeedFromOPML(c, folderID); err == nil {
				added++
			}
		}
	}
	return added, nil
}

func outlineName(o opmlOutline) string {
	if o.Title != "" {
		return o.Title
	}
	return o.Text
}

func (r *Repo) addFeedFromOPML(o opmlOutline, folderID *int) (Feed, error) {
	return r.CreateFeed(Feed{
		Title:    outlineName(o),
		FeedURL:  o.XMLURL,
		SiteURL:  o.HTMLURL,
		FolderID: folderID,
	})
}

func (r *Repo) ensureFolder(name string) (Folder, error) {
	if folder, err := r.CreateFolder(name); err == nil {
		return folder, nil
	}
	var f Folder
	err := r.DB.QueryRow(`SELECT id, name, sort_order FROM folders WHERE name = ?`, name).
		Scan(&f.ID, &f.Name, &f.SortOrder)
	return f, err
}

type opmlOutlineOut struct {
	Text    string           `xml:"text,attr"`
	Type    string           `xml:"type,attr"`
	XMLURL  string           `xml:"xmlUrl,attr"`
	HTMLURL string           `xml:"htmlUrl,attr,omitempty"`
	Outlines []opmlOutlineOut `xml:"outline,omitempty"`
}

type opmlDocOut struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    struct {
		Title string `xml:"title"`
	} `xml:"head"`
	Body struct {
		Outlines []opmlOutlineOut `xml:"outline"`
	} `xml:"body"`
}

// ExportOPML renders the current folders and feeds as an OPML 2.0 document.
func (r *Repo) ExportOPML() ([]byte, error) {
	folders, err := r.ListFolders()
	if err != nil {
		return nil, err
	}
	feeds, err := r.ListFeeds()
	if err != nil {
		return nil, err
	}
	var doc opmlDocOut
	doc.Version = "2.0"
	doc.Head.Title = "tinyrss"
	byFolder := map[int][]opmlOutlineOut{}
	var uncat []opmlOutlineOut
	for _, fd := range feeds {
		o := opmlOutlineOut{Text: fd.Title, Type: "rss", XMLURL: fd.FeedURL, HTMLURL: fd.SiteURL}
		if fd.FolderID == nil {
			uncat = append(uncat, o)
		} else {
			byFolder[*fd.FolderID] = append(byFolder[*fd.FolderID], o)
		}
	}
	doc.Body.Outlines = append(doc.Body.Outlines, uncat...)
	for _, folder := range folders {
		f := opmlOutlineOut{Text: folder.Name, Outlines: byFolder[folder.ID]}
		doc.Body.Outlines = append(doc.Body.Outlines, f)
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	buf.WriteString("\n")
	return buf.Bytes(), nil
}
